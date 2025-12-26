package dlfetch

import (
	"mime"
	"net/http"
	"os"
	"path/filepath"
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
