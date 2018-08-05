package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

type Merger struct {
	filePath   string
	channel    chan *Chunk
	file       *os.File
	waitGroup  *sync.WaitGroup
	read       int64
	totalSize  int64
	lastUpdate int64
	lastRead   int64
	speed      int64
}

type Chunk struct {
	data   []byte
	offset int64
}

func NewMerger(filePath string, totalSize int64) *Merger {
	m := new(Merger)
	m.filePath = filePath
	m.totalSize = totalSize
	m.channel = make(chan *Chunk, 128)
	m.waitGroup = new(sync.WaitGroup)
	m.read = 0
	m.lastUpdate = 0
	m.lastRead = 0
	m.speed = 0
	return m
}

func (m *Merger) Open() bool {
	file, err := os.Create(m.filePath)
	if err != nil {
		return false
	}

	m.file = file
	m.waitGroup.Add(1)

	return true
}

func (m *Merger) WriteAt(data []byte, offset int64) {
	m.channel <- &Chunk{data, offset}
}

// Block until the channel is empty
func (m *Merger) Close() {
	close(m.channel)
	m.waitGroup.Wait()
	m.logProgress()
}

func (m *Merger) Run() {
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

func (m *Merger) computeSpeed() {
	now := time.Now().UnixNano()

	if now - m.lastUpdate >= int64(time.Second) {
		m.speed = m.read - m.lastRead
		m.lastRead = m.read
		m.lastUpdate = now
	}
}

func (m *Merger) logProgress() {
	fmt.Printf("\r%.2f%% (%.3f MiB) @ %.3f MiB/s, %.2f%% buffer usage",
		100.0*float64(m.read)/float64(m.totalSize),
		float64(m.read)/toMiB,
		float64(m.speed)/toMiB,
		float32(len(m.channel))/128.0)
}
