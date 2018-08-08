package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	ErrFileInformationFetch    = errors.New("Could not fetch file information")
	ErrMultitthreadUnsupported = errors.New("Server does not support multithreaded downloads")
	ErrInvalidContentLength    = errors.New("Invalid content length received")
)

type ErrInvalidStatusCode struct {
	StatusCode int
}

func (e *ErrInvalidStatusCode) Error() string {
	return fmt.Sprintf("HTTP Status was: %i \n", e.StatusCode)
}

type downloader struct {
	url      string
	threads  int
	size     int64
	filePath string
	fileName string
	failure  error
}

type fileChunk struct {
	err   error
	buff  *bytes.Buffer
	start int64
	end   int64
}

func errChunk(err error) *fileChunk {
	return &fileChunk{
		err:   err,
		buff:  nil,
		start: 0,
		end:   0,
	}
}
func NewDownloader(url string, threads int) *downloader {

	path, _ := os.Getwd()

	return &downloader{
		url:      url,
		threads:  threads,
		filePath: path,
		fileName: "data",
	}

}

func (dl *downloader) Run() error {
	resp, err := http.Head(dl.url)
	if err != nil {
		fmt.Println("Could not fetch file information")
		return ErrFileInformationFetch
	}

	if resp.StatusCode != http.StatusOK {
		return &ErrInvalidStatusCode{StatusCode: resp.StatusCode}
	}

	if resp.Header.Get(headerAcceptRanges) != "bytes" {
		return ErrMultitthreadUnsupported
	}

	dl.size, err = strconv.ParseInt(resp.Header.Get(headerContentLength), 10, 64)
	if err != nil {
		fmt.Println("Invalid content length received")
		return ErrInvalidContentLength
	}

	dl.extractFileName()

	merger, err := newMerger(fmt.Sprintf("%s%c%s", dl.filePath, filepath.Separator, dl.fileName), dl.size)
	if err != nil {
		return fmt.Errorf("%s", err)
	}

	fmt.Printf("Starting %d threads\n", dl.threads)

	chunkSize := dl.size / int64(dl.threads)

	chunkChan := make(chan *fileChunk)
	go merger.Run(chunkChan)
	for i := 0; i < dl.threads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1

		// Don't forget the last bytes
		if i == dl.threads-1 {
			end = dl.size
		}

		go dl.downloadPart(i, start, end, chunkChan)
	}

	return nil
}

func (dl *downloader) extractFileName() {
	tokens := strings.Split(dl.url, "/")
	last := tokens[len(tokens)-1]
	if last != "" {
		dl.fileName = last
	}
}

func (dl *downloader) downloadPart(thread int, start int64, end int64, out chan<- *fileChunk) {

	req, err := http.NewRequest("GET", dl.url, nil)
	req.Header.Add(headerRange, fmt.Sprintf("bytes=%d-%d", start, end))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		out <- errChunk(err)
		return
	}

	if resp.StatusCode != http.StatusPartialContent {
		out <- errChunk(&ErrInvalidStatusCode{StatusCode: resp.StatusCode})
		return
	}

	chunk := &fileChunk{
		start: start,
		end:   end,
		buff:  &bytes.Buffer{},
		err:   nil,
	}

	defer resp.Body.Close()

	if _, err := io.Copy(chunk.buff, resp.Body); err != nil {
		out <- errChunk(err)
	} else {
		out <- chunk
	}

}
