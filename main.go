package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	if args == nil || len(args) == 0 {
		fmt.Println("Missing URL parameter")
		return
	}

	dl := NewDownloader(strings.Join(args, " "), 4)

	if err := dl.Run(); err != nil {
		fmt.Println("Download failed: ", err)
	}

}
