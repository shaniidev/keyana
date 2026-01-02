package download

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// FormatSize converts bytes to human-readable format
func FormatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	} else {
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	}
}

// GenerateFilename creates a safe filename from a URL
func GenerateFilename(rawURL string, index int) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		// Fallback to hash-based name
		hash := md5.Sum([]byte(rawURL))
		return fmt.Sprintf("file_%d_%x.js", index, hash[:8])
	}

	// Extract filename from path
	filename := path.Base(u.Path)

	// If no filename, use domain + index
	if filename == "" || filename == "/" || filename == "." {
		domain := u.Host
		domain = strings.ReplaceAll(domain, ".", "_")
		return fmt.Sprintf("%s_%d.js", domain, index)
	}

	// Ensure .js extension
	if !strings.HasSuffix(strings.ToLower(filename), ".js") {
		filename += ".js"
	}

	// Sanitize filename - remove/replace unsafe characters
	filename = sanitizeFilename(filename)

	// If filename is too long, truncate and add hash
	if len(filename) > 200 {
		hash := md5.Sum([]byte(rawURL))
		filename = filename[:180] + fmt.Sprintf("_%x.js", hash[:6])
	}

	return filename
}

// sanitizeFilename removes or replaces unsafe filename characters
func sanitizeFilename(name string) string {
	// Remove query parameters if accidentally included
	if idx := strings.Index(name, "?"); idx != -1 {
		name = name[:idx]
	}

	// Replace unsafe characters with underscore
	unsafe := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = unsafe.ReplaceAllString(name, "_")

	// Replace multiple underscores with single
	name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")

	// Trim leading/trailing dots and spaces
	name = strings.Trim(name, ". ")

	return name
}
