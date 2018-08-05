package main

import (
	"os"
	"strings"
	"fmt"
)

func main() {
	args := os.Args[1:]

	if args == nil || len(args) == 0 {
		fmt.Println("Missing URL parameter")
		return
	}

	dl := NewDownloader(strings.Join(args, " "), 4)
	if !dl.Run() {
		fmt.Println("Download failed !")
	}
}
