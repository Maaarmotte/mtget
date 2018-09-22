package main

import (
	"flag"
	"fmt"
	"mtget/mtget"
	"strings"
)

func main() {
	threadPtr := flag.Int("t", 4, "Number of threads")

	flag.Parse()

	args := flag.Args()

	if args == nil || len(args) == 0 {
		fmt.Println("Missing URL parameter")
		return
	}

	dl := mtget.NewDownloader(strings.Join(args, " "), *threadPtr)
	if !dl.Run() {
		fmt.Println("Download failed !")
	}
}
