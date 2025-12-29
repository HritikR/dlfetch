package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hritikr/dlfetch"
)

func main() {
	// Create a new fetcher with custom options
	var downloadMonitor dlfetch.Monitor = dlfetch.NewMonitor()
	fetcher := dlfetch.New(
		dlfetch.WithMaxWorkers(4),
		dlfetch.WithOnComplete(func(result dlfetch.DownloadResult) {
			fmt.Printf(
				"Download completed: id=%d file=%s path=%s mime=%s\n",
				result.ID,
				result.FileName,
				result.Path,
				result.MimeType,
			)
		}),
		dlfetch.WithOnError(func(req dlfetch.DownloadRequest, err error) {
			log.Printf(
				"Download failed: id=%d url=%s error=%v\n",
				req.ID,
				req.URL,
				err,
			)
		}),
		dlfetch.WithMonitor(downloadMonitor),
	)

	// Start worker pool
	fetcher.Start()

	// Enqueue a single download
	enqueueResult := fetcher.Enqueue(dlfetch.DownloadRequest{
		ID:  1,
		URL: "https://filesamples.com/samples/video/m4v/sample_3840x2160.m4v",
	})

	if enqueueResult.Error != nil {
		log.Printf("Failed to enqueue download: %v", enqueueResult.Error)
		fmt.Println("Nothing to download.")
		return
	}

	log.Println("Download queued successfully, waiting for completion...")

	// Check EventSignal to get updates on the download progress
	for {
		<-downloadMonitor.EventSignal()

		snapshot := downloadMonitor.GetSnapshot()
		data, err := json.MarshalIndent(snapshot, "", "  ")
		if err != nil {
			log.Printf("Error marshaling snapshot: %v", err)
		} else {
			log.Printf("=> Snapshot:\n%s\n", string(data))
		}

		// Exit on download complete
		if snapshot.Count.Completed+snapshot.Count.Failed == snapshot.Count.Total &&
			snapshot.Count.Total > 0 {
			break
		}
	}

	fmt.Println("All downloads processed.")

	// Stop the fetcher and wait for workers to exit
	fetcher.Stop()
}
