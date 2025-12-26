# dlfetch

dlfetch is a Go library designed to make file downloading easy and efficient. It uses Go's concurrency to handle multiple downloads, managed by a number of configurable workers. It also includes error handling and on completion callbacks.

## Usage Example

```go
package main

import "github.com/hritikr/dlfetch"

func main() {
    // Create a new Fetcher instance with default configuration
    fetcher := dlfetch.New()
    
    // Add a download request to the Fetcher's queue
    fetcher.Enqueue(dlfetch.DownloadRequest{
        ID: "example_file",
        URL: "https://example.com/file",
    })

    // Start processing download requests
    fetcher.Start()

    // When finished, stop the Fetcher
    fetcher.Stop()
}
```

For a more complete usage example, see the [example/example.go](example/example.go)
 in this repository.

## Functionality

dlfetch includes versatile configurability via functional options that allow you to:

* Change the default HTTP client
* Set the number of concurrent workers
* Specify the directory where downloaded files are saved
* Define custom behavior when a download completes or encounters an error

You can also add and manage multiple download requests at once using the `EnqueueMany()` function.

## Installation

```bash
go get -u github.com/hritikr/dlfetch
```

## Support & Contribution

For any issues or to provide feedback, please open an issue on this repository. We also welcome contributions. If you'd like to make changes or improve this library, please feel free to make a pull request.

This project is licensed under the MIT License. See LICENSE for more details.