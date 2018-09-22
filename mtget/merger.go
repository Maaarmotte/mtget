package mtget

import (
	"fmt"
	"math"
	"os"
	"sync"
	"time"
)

const (
	preallocationBufferSize = 10485760
	maxWaitingChunks = 512
)

type merger struct {
	filePath   string
	channel    chan *chunk
	file       *os.File
	waitGroup  *sync.WaitGroup
	read       int64
	totalSize  int64
	lastUpdate int64
	lastRead   int64
	speed      int64
}

type chunk struct {
	data   []byte
	offset int64
}

func newMerger(filePath string, totalSize int64) *merger {
	m := new(merger)
	m.filePath = filePath
	m.totalSize = totalSize
	m.channel = make(chan *chunk, maxWaitingChunks)
	m.waitGroup = new(sync.WaitGroup)
	return m
}

func (m *merger) Open() bool {
	file, err := os.Create(m.filePath)
	if err != nil {
		return false
	}

	m.file = file
	m.waitGroup.Add(1)

	return true
}

// Preallocate space to avoid unwanted freeze
func (m *merger) Preallocate() {
	arr := make([]byte, preallocationBufferSize)
	pos := int64(0)

	for {
		if pos + preallocationBufferSize > m.totalSize {
			m.file.WriteAt(arr[0:m.totalSize - pos], pos)
		} else {
			m.file.WriteAt(arr, pos)
		}
		pos += preallocationBufferSize

		fmt.Printf("\rPreallocating... %.2f%%", 100.0*math.Min(float64(pos), float64(m.totalSize))/float64(m.totalSize))

		if pos >= m.totalSize {
			fmt.Println()
			break
		}
	}
}

func (m *merger) WriteAt(data []byte, offset int64) {
	m.channel <- &chunk{data, offset}
}

// Block until the channel is empty
func (m *merger) Close() {
	close(m.channel)
	m.waitGroup.Wait()
	m.logProgress()
}

func (m *merger) Run() {
	defer m.waitGroup.Done()
	defer m.file.Close()

	for {
		chunk := <-m.channel

		// Channel is closed
		if chunk == nil {
			break
		}

		m.file.WriteAt(chunk.data, chunk.offset)
		m.read += int64(len(chunk.data))

		m.computeSpeed()
		m.logProgress()
	}
}

func (m *merger) computeSpeed() {
	now := time.Now().UnixNano()

	if now - m.lastUpdate >= int64(time.Second) {
		m.speed = m.read - m.lastRead
		m.lastRead = m.read
		m.lastUpdate = now
	}
}

func (m *merger) logProgress() {
	fmt.Printf("\r%.2f%% (%.3f MiB) @ %.3f MiB/s, %.2f%% buffer usage     ",
		100.0*float64(m.read)/float64(m.totalSize),
		float64(m.read)/toMiB,
		float64(m.speed)/toMiB,
		float32(100.0*len(m.channel))/maxWaitingChunks)
}
