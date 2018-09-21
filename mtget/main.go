package main

import (
	"fmt"
	"mtget/mtgetlib"
	"os"
	"strings"
)

func main() {
	args := os.Args[1:]

	if args == nil || len(args) == 0 {
		fmt.Println("Missing URL parameter")
		return
	}

	dl := mtgetlib.NewDownloader(strings.Join(args, " "), 16)
	if !dl.Run() {
		fmt.Println("Download failed !")
	}
}
