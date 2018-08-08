package main

import (
	"fmt"
	"os"
	"time"
)

type merger struct {
	file       *os.File
	read       int64
	totalSize  int64
	lastUpdate time.Time
	lastRead   int64
	speed      int64
}

func newMerger(filePath string, totalSize int64) (*merger, error) {

	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	return &merger{
		file:      file,
		totalSize: totalSize,
	}, nil
}

func (m *merger) Run(in <-chan *fileChunk) {
	defer m.file.Close()

	for chunk := range in {
		if chunk.err != nil {
			break
		} else {
			if _, err := m.file.WriteAt(chunk.buff.Bytes(), chunk.start); err != nil {
				break
			} else {
				m.read += int64(chunk.buff.Len())
				m.computeSpeed()
				m.logProgress(in)
			}
		}
	}
}

func (m *merger) computeSpeed() {
	now := time.Now()

	if now.Sub(m.lastUpdate) >= time.Second {
		m.speed = m.read - m.lastRead
		m.lastRead = m.read
		m.lastUpdate = now
	}
}

func (m *merger) logProgress(in <-chan *fileChunk) {
	fmt.Printf("\r%.2f%% (%.3f MiB) @ %.3f MiB/s, %.2f%% buffer usage",
		100.0*float64(m.read)/float64(m.totalSize),
		float64(m.read)/toMiB,
		float64(m.speed)/toMiB,
		float32(len(in)/cap(in)))
}
