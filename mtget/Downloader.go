package main

import (
	"net/http"
	"fmt"
	"strconv"
	"os"
	"bufio"
	"io"
	"sync"
	"path/filepath"
	"strings"
)

type Downloader struct {
	url       string
	threads   int
	size      int64
	filePath  string
	fileName  string
	waitGroup *sync.WaitGroup
	failure   bool
	merger    *Merger
}

func NewDownloader(url string, threads int) *Downloader {
	dl := new(Downloader)
	dl.url = url
	dl.threads = threads
	dl.filePath, _ = os.Getwd()
	dl.fileName = "data"
	dl.waitGroup = new(sync.WaitGroup)
	dl.failure = false
	return dl
}

func (dl *Downloader) Run() bool {
	resp, err := http.Head(dl.url)
	if err != nil {
		fmt.Println("Could not fetch file information")
		return false
	}

	if resp.StatusCode != statusOk {
		fmt.Println("HTTP Status was", resp.Status)
		return false
	}

	if resp.Header.Get(headerAcceptRanges) != "bytes" {
		fmt.Println("Server does not support multithreaded downloads")
		return false
	}

	dl.size, err = strconv.ParseInt(resp.Header.Get(headerContentLength), 10, 64)
	if err != nil {
		fmt.Println("Invalid content length received")
		return false
	}

	dl.extractFileName()
	dl.merger = NewMerger(fmt.Sprintf("%s%c%s", dl.filePath, filepath.Separator, dl.fileName), dl.size)
	if !dl.merger.Open() {
		fmt.Println("Could not create output file")
	}

	fmt.Printf("Starting %d threads\n", dl.threads)

	chunkSize := dl.size / int64(dl.threads)
	for i := 0; i < dl.threads; i++ {
		start := int64(i) * chunkSize
		end := start + chunkSize - 1

		// Don't forget the last bytes
		if i == dl.threads-1 {
			end = dl.size
		}

		go dl.downloadPart(i, start, end)
	}

	go dl.merger.Run()

	dl.waitGroup.Add(dl.threads)
	dl.waitGroup.Wait()

	dl.merger.Close()

	return !dl.failure
}

func (dl *Downloader) extractFileName() {
	tokens := strings.Split(dl.url, "/")
	last := tokens[len(tokens) - 1]
	if last != "" {
		dl.fileName = last
	}
}

func (dl *Downloader) downloadPart(thread int, start int64, end int64) {
	defer dl.waitGroup.Done()
	defer dl.safeStop(thread)

	req, err := http.NewRequest("GET", dl.url, nil)
	req.Header.Add(headerRange, fmt.Sprintf("bytes=%d-%d", start, end))

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		dl.fail(err)
	}

	if resp.StatusCode != statusPartialContent {
		fmt.Printf("%d] HTTP Status was %s\n", thread, resp.Status)
		dl.failure = true
		return
	}

	reader := bufio.NewReader(resp.Body)
	pos := start

	for {
		if dl.failure {
			fmt.Printf("%d] Download aborted\n", thread)
			return
		}

		buffer := make([]byte, 16384)
		count, err := reader.Read(buffer)

		if count > 0 {
			dl.merger.WriteAt(buffer[:count], pos)
			pos += int64(count)
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			fmt.Printf("%d] Error while downloading file\n", thread)
			dl.fail(err)
		}
	}
}

func (dl *Downloader) fail(err error) {
	dl.failure = true
	panic(err)
}

func (dl *Downloader) safeStop(thread int) {
	r := recover()
	if r != nil {
		fmt.Printf("%d] Download aborted\n", thread)
	}
}
