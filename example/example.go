package main

import (
	"fmt"
	"log"
	"time"

	"github.com/hritikr/dlfetch"
)

func main() {
	// Create a new fetcher with custom options
	fetcher := dlfetch.New(
		dlfetch.WithMaxWorkers(4),
		dlfetch.WithOnComplete(func(result dlfetch.DownloadResult) {
			fmt.Printf(
				"Download completed: id=%s file=%s path=%s mime=%s\n",
				result.ID,
				result.FileName,
				result.Path,
				result.MimeType,
			)
		}),
		dlfetch.WithOnError(func(req dlfetch.DownloadRequest, err error) {
			log.Printf(
				"Download failed: id=%s url=%s error=%v\n",
				req.ID,
				req.URL,
				err,
			)
		}),
	)

	// Start worker pool
	fetcher.Start()

	// Enqueue a single download
	fetcher.Enqueue(dlfetch.DownloadRequest{
		ID:  1,
		URL: "https://github.com/karuncodes/sample-files/raw/refs/heads/main/sample%20video/sample.mp4",
	})

	// Wait for downloads to finish
	time.Sleep(10 * time.Second)

	// Stop the fetcher and wait for workers to exit
	fetcher.Stop()

	fmt.Println("All downloads processed.")
}
