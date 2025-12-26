package dlfetch

import (
	"mime"
	"net/http"
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
	return os.MkdirAll(dir, 0755)
}

// DetermineMimeType returns the most accurate MIME type for a downloaded file.
func DetermineMimeType(req DownloadRequest, respContentType string, filePath string) string {
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
