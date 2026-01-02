package beautify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"keyana/internal/config"
	"keyana/internal/core"
	"keyana/internal/ui"
)

type Beautifier struct {
	Config *config.Config
}

func NewBeautifier(cfg *config.Config) *Beautifier {
	return &Beautifier{Config: cfg}
}

func (b *Beautifier) Run(files []*core.JSFile) []string {
	var beautifiedPaths []string
	var mu sync.Mutex

	beautifiedDir := filepath.Join(b.Config.OutputDir, "beautified")
	os.MkdirAll(beautifiedDir, 0755)

	// Create log file for beautification errors (Append Mode)
	logPath := filepath.Join(b.Config.OutputDir, "logs", "beautify.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	// Filter out already beautified files
	var filesToBeautify []*core.JSFile
	skipped := 0

	for _, f := range files {
		outPath := filepath.Join(beautifiedDir, f.Filename)
		if info, err := os.Stat(outPath); err == nil && info.Size() > 0 {
			// Already beautified, add to result and skip
			skipped++
			beautifiedPaths = append(beautifiedPaths, outPath)
			f.Beautified = true
		} else {
			filesToBeautify = append(filesToBeautify, f)
		}
	}

	if skipped > 0 {
		ui.Printf(ui.Green, "[+] Skipping %d already beautified files\n", skipped)
	}

	if len(filesToBeautify) == 0 {
		ui.Success("All files already beautified!")
		return beautifiedPaths
	}

	sem := make(chan struct{}, b.Config.Concurrency)
	var wg sync.WaitGroup

	bar := ui.NewProgressBar(len(filesToBeautify), "Beautifying")

	for _, file := range filesToBeautify {
		wg.Add(1)
		go func(f *core.JSFile) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Update Bar
			defer bar.Increment()

			// Input path
			inPath := f.LocalPath
			// Output path
			outPath := filepath.Join(beautifiedDir, f.Filename)

			err := b.beautifyFile(inPath, outPath)
			if err == nil {
				mu.Lock()
				beautifiedPaths = append(beautifiedPaths, outPath)
				f.Beautified = true
				mu.Unlock()
			} else {
				// Log beautification failure (Rule 10: Log Everything)
				mu.Lock()
				if logFile != nil {
					fmt.Fprintf(logFile, "[ERROR] %s: %v\n", f.Filename, err)
				}
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()

	if skipped > 0 {
		ui.Printf(ui.Cyan, "[*] Resumed: %d files from previous session\n", skipped)
	}

	return beautifiedPaths
}

func (b *Beautifier) beautifyFile(inPath, outPath string) error {
	// js-beautify <in> -o <out>
	cmd := exec.Command("js-beautify", inPath, "-o", outPath)
	return cmd.Run()
}
