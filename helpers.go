package dlfetch

import (
	"os"
	"path/filepath"
)

// CheckFileExists checks if a file exists at the given path.
func CheckFileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// EnsureFileName ensures that the DownloadRequest has a valid FileName.
// If FileName is empty, it extracts the file name from the URL.
func EnsureFileName(req *DownloadRequest) {
	if req.FileName != "" {
		return
	}
	req.FileName = filepath.Base(req.URL)
}

// EnsureDir ensures that the directory for the given path exists.
func EnsureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, os.ModePerm)
}
