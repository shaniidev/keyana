package download

import (
	"bufio"
	"fmt"
	"keyana/internal/config"
	"keyana/internal/core"
	"keyana/internal/ui"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Downloader struct {
	Config        *config.Config
	downloadedMap map[string]string // URL -> filename mapping
	mapMu         sync.RWMutex
}

func NewDownloader(cfg *config.Config) *Downloader {
	d := &Downloader{
		Config:        cfg,
		downloadedMap: make(map[string]string),
	}
	d.loadDownloadedURLs()
	return d
}

// loadDownloadedURLs loads previously downloaded URLs from mapping file
func (d *Downloader) loadDownloadedURLs() {
	mapPath := filepath.Join(d.Config.OutputDir, "js_files", "downloaded_urls.txt")
	file, err := os.Open(mapPath)
	if err != nil {
		return // File doesn't exist yet, that's fine
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Format: URL|filename
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			d.downloadedMap[parts[0]] = parts[1]
		}
	}
}

// saveDownloadedURL appends a URL->filename mapping to the tracking file
func (d *Downloader) saveDownloadedURL(url, filename string) {
	d.mapMu.Lock()
	defer d.mapMu.Unlock()

	d.downloadedMap[url] = filename

	mapPath := filepath.Join(d.Config.OutputDir, "js_files", "downloaded_urls.txt")
	f, err := os.OpenFile(mapPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintf(f, "%s|%s\n", url, filename)
}

// isAlreadyDownloaded checks if URL was previously downloaded and file exists
func (d *Downloader) isAlreadyDownloaded(url string) (bool, string) {
	d.mapMu.RLock()
	filename, exists := d.downloadedMap[url]
	d.mapMu.RUnlock()

	if !exists {
		return false, ""
	}

	// Verify file still exists
	rawDir := filepath.Join(d.Config.OutputDir, "js_files", "raw")
	filePath := filepath.Join(rawDir, filename)
	if info, err := os.Stat(filePath); err == nil && info.Size() > 0 {
		return true, filename
	}

	return false, ""
}

// Run downloads all JS files concurrently and returns the results
func (d *Downloader) Run(urls []string) []*core.JSFile {
	if len(urls) == 0 {
		return []*core.JSFile{}
	}

	// Output directory for raw files
	rawDir := filepath.Join(d.Config.OutputDir, "js_files", "raw")

	// Create log file for detailed output (Append Mode)
	logPath := filepath.Join(d.Config.OutputDir, "logs", "download.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	// Filter out already downloaded URLs
	var urlsToDownload []string
	var alreadyDownloaded []*core.JSFile
	skipped := 0

	for _, url := range urls {
		if exists, filename := d.isAlreadyDownloaded(url); exists {
			skipped++
			alreadyDownloaded = append(alreadyDownloaded, &core.JSFile{
				URL:        url,
				Filename:   filename,
				LocalPath:  filepath.Join(rawDir, filename),
				Downloaded: true,
			})
		} else {
			urlsToDownload = append(urlsToDownload, url)
		}
	}

	if skipped > 0 {
		ui.Printf(ui.Green, "[+] Skipping %d already downloaded files (resume mode)\n", skipped)
	}

	if len(urlsToDownload) == 0 {
		ui.Success("All files already downloaded!")
		return alreadyDownloaded
	}

	// Create channels for job distribution
	jobs := make(chan downloadJob, len(urlsToDownload))
	results := make(chan *core.JSFile, len(urlsToDownload))

	// Statistics
	var mu sync.Mutex
	var logMu sync.Mutex
	stats := &downloadStats{
		total:   len(urlsToDownload),
		success: 0,
		failed:  0,
	}

	// Start worker pool
	var wg sync.WaitGroup
	numWorkers := d.Config.Concurrency
	if numWorkers <= 0 {
		numWorkers = 20
	}

	// Create progress bar
	bar := ui.NewProgressBar(len(urlsToDownload), "Downloading")

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go d.worker(i, jobs, results, &wg, rawDir, stats, &mu, logFile, &logMu, bar)
	}

	// Send all jobs to workers
	for idx, url := range urlsToDownload {
		jobs <- downloadJob{
			url:   url,
			index: skipped + idx + 1, // Continue numbering from skipped files
		}
	}
	close(jobs)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var jsFiles []*core.JSFile
	jsFiles = append(jsFiles, alreadyDownloaded...) // Add previously downloaded files
	for jsFile := range results {
		jsFiles = append(jsFiles, jsFile)
	}

	// Print final statistics
	fmt.Println()
	ui.Printf(ui.Green, "[+] Download Complete: %d/%d successful\n", stats.success, stats.total)
	if stats.failed > 0 {
		ui.Printf(ui.Yellow, "[!] Failed: %d files\n", stats.failed)
	}
	if skipped > 0 {
		ui.Printf(ui.Cyan, "[*] Resumed: %d files from previous session\n", skipped)
	}
	fmt.Printf("[+] Download log saved: %s\n", logPath)

	return jsFiles
}

// worker processes download jobs from the channel
func (d *Downloader) worker(id int, jobs <-chan downloadJob, results chan<- *core.JSFile, wg *sync.WaitGroup, rawDir string, stats *downloadStats, mu *sync.Mutex, logFile *os.File, logMu *sync.Mutex, bar *ui.ProgressBar) {
	defer wg.Done()

	for job := range jobs {
		// Generate filename
		filename := GenerateFilename(job.url, job.index)
		outputPath := filepath.Join(rawDir, filename)

		// Attempt download
		success, statusCode, size, err := DownloadFile(job.url, outputPath, d.Config.Timeout, 0)

		// Update statistics
		mu.Lock()
		if success {
			stats.success++
			stats.totalBytes += size
			// Save to mapping file for resume capability
			d.saveDownloadedURL(job.url, filename)
		} else {
			stats.failed++
		}
		current := stats.success + stats.failed
		mu.Unlock()

		// Create JSFile result
		jsFile := &core.JSFile{
			URL:        job.url,
			Filename:   filename,
			LocalPath:  outputPath,
			Downloaded: success,
		}

		// Log detailed info to file (thread-safe)
		logMu.Lock()
		if success {
			fmt.Fprintf(logFile, "[%d/%d] SUCCESS: %s (%s) - %s\n",
				current, stats.total,
				filename,
				FormatSize(size),
				job.url)
		} else {
			errMsg := "failed"
			if err != nil {
				errMsg = err.Error()
			}
			fmt.Fprintf(logFile, "[%d/%d] FAILED: %s (HTTP %d: %s) - %s\n",
				current, stats.total,
				filename,
				statusCode,
				errMsg,
				job.url)
		}
		logMu.Unlock()

		// Update progress bar
		bar.Increment()

		results <- jsFile
	}
}

// downloadJob represents a single download task
type downloadJob struct {
	url   string
	index int
}

// downloadStats tracks download statistics
type downloadStats struct {
	total      int
	success    int
	failed     int
	totalBytes int64
}
