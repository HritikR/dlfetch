package dlfetch

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// checkFileExists checks if a file exists at the given path.
func checkFileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// ensureFileName ensures that the DownloadRequest has a valid FileName.
// If FileName is empty, it extracts the file name from the URL.
func ensureFileName(req *DownloadRequest) {
	if req.FileName != "" {
		return
	}
	req.FileName = filepath.Base(req.URL)
}

// ensureDir ensures that the directory for the given path exists.
func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}

// determineMimeType returns the most accurate MIME type for a downloaded file.
func determineMimeType(req DownloadRequest, respContentType string, filePath string) string {
	if respContentType != "" && respContentType != "application/octet-stream" {
		return respContentType
	}
	if req.MimeType != "" {
		return req.MimeType
	}
	if ext := filepath.Ext(filePath); ext != "" {
		if mt := mime.TypeByExtension(ext); mt != "" {
			return mt
		}
	}
	// fallback: detect from file bytes
	file, _ := os.Open(filePath)
	defer file.Close()
	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	return http.DetectContentType(buf[:n])
}

// isOfType is a generic helper to check if the file belongs to a category like "image", "video", "audio"
func (d *DownloadResult) isOfType(category string) bool {
	category = strings.ToLower(category)

	// Check MIME type first
	if d.MimeType != "" && strings.HasPrefix(strings.ToLower(d.MimeType), category+"/") {
		return true
	}

	// Fallback: check by file extension
	ext := strings.ToLower(filepath.Ext(d.FileName))
	mimeType := mime.TypeByExtension(ext)
	return strings.HasPrefix(mimeType, category+"/")
}

func (d *DownloadResult) IsImage() bool {
	return d.isOfType("image")
}

func (d *DownloadResult) IsVideo() bool {
	return d.isOfType("video")
}

func (d *DownloadResult) IsAudio() bool {
	return d.isOfType("audio")
}

// validateRequest checks if the file name is not nil or empty
// also checks if file already exists
func (f *Fetcher) validateRequest(req *DownloadRequest) error {
	ensureFileName(req)

	req.FullPath = filepath.Join(f.targetDir, req.Path, req.FileName)

	if checkFileExists(req.FullPath) {
		return fmt.Errorf("file already exists: %s", req.FullPath)
	}

	return nil
}

// resolveFileSize attempts to find the file size from various headers.
// Returns -1 if the size cannot be determined (e.g., chunked transfer).
func resolveFileSize(resp *http.Response) int64 {
	if resp.ContentLength > 0 {
		return resp.ContentLength
	}

	// Checking content-length manually in case of the "Transfer-Encoding: chunked" scenario
	if cl := resp.Header.Get("Content-Length"); cl != "" {
		if size, err := strconv.ParseInt(strings.TrimSpace(cl), 10, 64); err == nil && size > 0 {
			return size
		}
	}

	// Checkinng Content-Range
	if cr := resp.Header.Get("Content-Range"); cr != "" {
		// Handle both "bytes 0-999/1000" and "bytes */1000"
		if idx := strings.LastIndex(cr, "/"); idx != -1 {
			totalStr := strings.TrimSpace(cr[idx+1:])
			if totalStr != "*" {
				if size, err := strconv.ParseInt(totalStr, 10, 64); err == nil && size > 0 {
					return size
				}
			}
		}
	}

	// Unknown size
	return -1
}
