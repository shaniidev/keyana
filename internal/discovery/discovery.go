package discovery

import (
	"bufio"
	"fmt"
	"keyana/internal/config"
	"keyana/internal/ui"
	"keyana/internal/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type DiscoveryManager struct {
	Config *config.Config
}

// toolStats tracks discovery stats for each tool
type toolStats struct {
	name    string
	total   int64
	jsCount int64
	done    int32
}

func NewDiscoveryManager(cfg *config.Config) *DiscoveryManager {
	return &DiscoveryManager{Config: cfg}
}

// Run executes all discovery tools concurrently with live counter display
func (dm *DiscoveryManager) Run() []string {
	var wg sync.WaitGroup
	results := make(chan string, 100000)

	// Create urls directory
	urlsDir := filepath.Join(dm.Config.OutputDir, "urls")
	os.MkdirAll(urlsDir, 0755)

	tools := []struct {
		Name string
		File string
	}{
		{"katana", "katana_urls.txt"},
		{"gau", "gau_urls.txt"},
		{"waybackurls", "wayback_urls.txt"},
	}

	// Initialize stats for each tool
	stats := make([]*toolStats, len(tools))
	for i, t := range tools {
		stats[i] = &toolStats{name: t.Name}
	}

	// Start display updater
	var displayWg sync.WaitGroup
	displayWg.Add(1)
	displayDone := make(chan struct{})
	go func() {
		defer displayWg.Done()
		dm.displayUpdater(stats, displayDone)
	}()

	for i, t := range tools {
		wg.Add(1)
		go func(idx int, name, fname string) {
			defer wg.Done()
			defer func() { atomic.StoreInt32(&stats[idx].done, 1) }()
			dm.runTool(name, filepath.Join(urlsDir, fname), results, stats[idx])
		}(i, t.Name, t.File)
	}

	// Closer routine
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and deduplicate
	seen := make(map[string]bool)
	var finalURLs []string

	for url := range results {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		// Filter: Process only JS files
		if !utils.IsJSFile(url) {
			continue
		}

		if !seen[url] {
			seen[url] = true
			finalURLs = append(finalURLs, url)
		}
	}

	// Signal display to finish
	close(displayDone)
	displayWg.Wait() // Wait for display to complete

	// Print final summary
	fmt.Printf("\n\n")
	ui.Success("Discovery complete: %d unique JS files found", len(finalURLs))

	return finalURLs
}

// displayUpdater refreshes the counter display every 200ms
func (dm *DiscoveryManager) displayUpdater(stats []*toolStats, done chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	// Print initial state
	fmt.Println()
	for range stats {
		fmt.Println() // Reserve lines
	}

	for {
		select {
		case <-done:
			// Drain any pending ticker events to prevent duplicate prints
			select {
			case <-ticker.C:
			default:
			}
			// Print final stats once
			dm.printStats(stats, true)
			return
		case <-ticker.C:
			dm.printStats(stats, false)
		}
	}
}

// printStats prints the current stats for all tools
func (dm *DiscoveryManager) printStats(stats []*toolStats, final bool) {
	// Move cursor up to overwrite previous lines
	fmt.Printf("\033[%dA", len(stats))

	// Spinner frames for activity indication
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	spinnerIdx := int(time.Now().UnixMilli()/100) % len(spinnerFrames)

	for _, s := range stats {
		total := atomic.LoadInt64(&s.total)
		jsCount := atomic.LoadInt64(&s.jsCount)
		isDone := atomic.LoadInt32(&s.done) == 1

		var status string
		if isDone {
			status = ui.Green + "✓" + ui.Reset
		} else {
			status = ui.Yellow + spinnerFrames[spinnerIdx] + ui.Reset
		}

		// Clear line and print (no fake progress bar)
		fmt.Printf("\r\033[K%s [%-12s] %7d URLs  |  %s%d JS%s\n",
			status,
			s.name,
			total,
			ui.Green,
			jsCount,
			ui.Reset,
		)
	}
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (dm *DiscoveryManager) runTool(toolName, outFile string, out chan<- string, stats *toolStats) {
	var cmd *exec.Cmd
	target := dm.Config.Domain

	// Construct commands
	switch toolName {
	case "katana":
		args := []string{"-u", target, "-d", "5", "-jc", "-silent"}
		cmd = exec.Command("katana", args...)
	case "gau":
		cmd = exec.Command("gau", target, "--subs")
	case "waybackurls":
		cmd = exec.Command("waybackurls", target)
	}

	if cmd == nil {
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logDiscoveryError(dm.Config.OutputDir, toolName, "StdoutPipe failed", err)
		return
	}

	if err := cmd.Start(); err != nil {
		logDiscoveryError(dm.Config.OutputDir, toolName, "Start failed", err)
		return
	}

	// Create file to save raw tool output (Append Mode - Rule 5)
	f, err := os.OpenFile(outFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		logPath := filepath.Join(filepath.Dir(outFile), "..", "logs", "discovery.log")
		if logFile, lerr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); lerr == nil {
			fmt.Fprintf(logFile, "[ERROR] Failed to create output file %s: %v\n", outFile, err)
			logFile.Close()
		}
	} else {
		defer f.Close()
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()

		// Update stats
		atomic.AddInt64(&stats.total, 1)
		if utils.IsJSFile(line) {
			atomic.AddInt64(&stats.jsCount, 1)
		}

		// Write to raw file if open
		if f != nil {
			f.WriteString(line + "\n")
		}

		// Send to aggregator
		out <- line
	}

	cmd.Wait()
}

// logDiscoveryError logs discovery tool errors to discovery.log
func logDiscoveryError(outputDir, toolName, context string, err error) {
	logPath := filepath.Join(outputDir, "logs", "discovery.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, lerr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if lerr != nil {
		return
	}
	defer logFile.Close()
	fmt.Fprintf(logFile, "[ERROR] %s: %s - %v\n", toolName, context, err)
}
