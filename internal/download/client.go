package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DownloadFile downloads a file from URL with retry logic and rate limiting
// Returns: success, statusCode, sizeBytes, error
func DownloadFile(url, outputPath string, timeout int, retry int) (bool, int, int64, error) {
	// Connection timeout only - downloads complete regardless of size
	connectionTimeout := 30 * time.Second
	if timeout > 30 {
		connectionTimeout = time.Duration(timeout) * time.Second
	}

	// Create HTTP client with connection timeout only
	client := &http.Client{
		Timeout: 0, // No overall timeout - let files download completely
		Transport: &http.Transport{
			DisableKeepAlives:     false,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: connectionTimeout, // Timeout for server to respond, not download
			ExpectContinueTimeout: 1 * time.Second,
		},
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, 0, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to mimic browser behavior
	// Do NOT set Accept-Encoding manually; Go client handles gzip automatically
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/javascript, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Connection", "keep-alive")

	resp, err := client.Do(req)
	if err != nil {
		// Retry on connection error
		if retry < 2 {
			time.Sleep(time.Duration(2<<uint(retry)) * time.Second) // Exponential backoff
			return DownloadFile(url, outputPath, timeout, retry+1)
		}
		return false, 0, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode

	// Handle rate limiting and server errors with retry
	if (statusCode == 429 || statusCode == 503 || statusCode == 504) && retry < 2 {
		backoff := time.Duration(1<<uint(retry)) * time.Second // Exponential backoff: 2^retry
		time.Sleep(backoff)
		return DownloadFile(url, outputPath, timeout, retry+1)
	}

	// Only accept successful status codes
	if statusCode < 200 || statusCode >= 300 {
		// Special handling for 304 Not Modified
		if statusCode == 304 {
			return true, statusCode, 0, nil
		}
		return false, statusCode, 0, fmt.Errorf("HTTP %d", statusCode)
	}

	// Create output file (Prevent Overwrite - Rule 7)
	// Check if file exists and is not empty
	if info, err := os.Stat(outputPath); err == nil && info.Size() > 0 {
		return true, 304, info.Size(), nil // Treat as "Not Modified" / Already done
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return false, statusCode, 0, fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Download file content - no timeout, will complete no matter how long it takes
	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		os.Remove(outputPath) // Clean up partial file
		return false, statusCode, 0, fmt.Errorf("failed to write file: %w", err)
	}

	// Verify file size
	if written == 0 {
		os.Remove(outputPath) // Clean up empty file
		return false, statusCode, 0, fmt.Errorf("empty file downloaded")
	}

	return true, statusCode, written, nil
}
