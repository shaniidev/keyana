package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shaniidev/keyana/internal/beautify"
	"github.com/shaniidev/keyana/internal/config"
	"github.com/shaniidev/keyana/internal/core"
	"github.com/shaniidev/keyana/internal/discovery"
	"github.com/shaniidev/keyana/internal/download"
	"github.com/shaniidev/keyana/internal/scan"
	"github.com/shaniidev/keyana/internal/ui"
	"github.com/shaniidev/keyana/internal/utils"
)

func main() {
	// Pre-flight check
	checkDependencies()

	cfg := config.NewConfig()
	cfg.ParseFlags()

	if !cfg.Silent {
		printBanner()
		// Load secret detection patterns at startup
		loadTemplatePatterns()
	}

	if cfg.Domain == "" && cfg.ListFile == "" && cfg.URLsFile == "" && cfg.RawDir == "" && cfg.BeautifiedDir == "" {
		fmt.Println("Error: usage: keyana -d example.com [flags]")
		fmt.Println("       or: keyana -urls urls.txt")
		fmt.Println("       or: keyana -raw ./js_raw/")
		fmt.Println("       or: keyana -beautified ./js_beautified/")
		os.Exit(1)
	}

	state := core.NewPipelineState()
	fmt.Printf("[*] Target: %s\n", cfg.Domain)
	fmt.Printf("[*] Output: %s\n", cfg.OutputDir)

	// Create folder structure
	os.MkdirAll(filepath.Join(cfg.OutputDir, "reports"), 0755)
	os.MkdirAll(filepath.Join(cfg.OutputDir, "logs"), 0755)
	os.MkdirAll(filepath.Join(cfg.OutputDir, "urls"), 0755)
	os.MkdirAll(filepath.Join(cfg.OutputDir, "js_files", "raw"), 0755)
	// Beautified folder created by beautifier at OutputDir/beautified

	// ---------------------------------------------------------
	// STAGE 1: DISCOVERY (OR LOAD URLs)
	// ---------------------------------------------------------
	// ---------------------------------------------------------
	// STAGE 1: DISCOVERY (OR LOAD URLs)
	// ---------------------------------------------------------
	if cfg.RawDir == "" && cfg.BeautifiedDir == "" {
		urls := runDiscoveryStage(cfg)
		state.URLs = uniqueAPI(urls)
	} else {
		fmt.Println("[*] Skipping Discovery Stage (Input provided for later stages)")
	}

	// ---------------------------------------------------------
	// STAGE 2: DOWNLOAD (OR LOAD RAW FILES)
	// ---------------------------------------------------------
	if cfg.BeautifiedDir == "" {
		state.RawJSFiles = runDownloadStage(cfg, state.URLs)
	} else {
		fmt.Println("[*] Skipping Download Stage (Beautified Input provided)")
	}

	// ---------------------------------------------------------
	// STAGE 3: BEAUTIFICATION (OR LOAD BEAUTIFIED FILES)
	// ---------------------------------------------------------
	var scanFiles []string
	if cfg.BeautifiedDir != "" {
		scanFiles = loadBeautifiedFiles(cfg)
	} else {
		scanFiles = runBeautifyStage(cfg, state.RawJSFiles)
	}
	state.BeautifiedFiles = scanFiles

	if len(scanFiles) == 0 {
		// No files found to scan
	}

	scanChoice := ui.PromptScanChoice(len(scanFiles))

	// ---------------------------------------------------------
	// STAGE 4 & 5: SCANNING
	// ---------------------------------------------------------
	runScanStage(cfg, state, scanChoice, scanFiles)

	fmt.Println("\n[+] KEYANA Finished. Check output directory.")
}

// runDiscoveryStage handles logic for URL collection
func runDiscoveryStage(cfg *config.Config) []string {
	var urls []string
	shouldRunDiscovery := true

	// Check for existing discovery files in the urls directory
	urlsDir := filepath.Join(cfg.OutputDir, "urls")
	internalFiles := []string{"katana_urls.txt", "gau_urls.txt", "wayback_urls.txt"}
	foundExisting := false

	// If we are NOT forcing an external file (-urls), check internal
	if cfg.URLsFile == "" {
		for _, f := range internalFiles {
			if _, err := os.Stat(filepath.Join(urlsDir, f)); err == nil {
				foundExisting = true
				break
			}
		}

		if foundExisting {
			fmt.Println("\n[!] Found existing discovery data in output directory.")
			ans := ui.Prompt("    Load these instead of re-running discovery? (Y/n) [Y]:")
			if ans == "" || strings.ToLower(ans) == "y" {
				shouldRunDiscovery = false
				fmt.Println("[*] Loading existing discovery files...")
				for _, f := range internalFiles {
					fpath := filepath.Join(urlsDir, f)
					if file, err := os.Open(fpath); err == nil {
						scanner := bufio.NewScanner(file)
						count := 0
						for scanner.Scan() {
							line := scanner.Text()
							if line != "" {
								// Fix: Filter only JS files when loading from existing data
								if !utils.IsJSFile(line) {
									continue
								}
								urls = append(urls, line)
								count++
							}
						}
						file.Close()
						if count > 0 {
							fmt.Printf("    Loaded %d URLs from %s\n", count, f)
						}
					}
				}
			}
		}
	}

	if cfg.URLsFile != "" {
		fmt.Printf("\n[STAGE 1] Loading URLs from %s (Skipping Discovery)\n", cfg.URLsFile)
		file, err := os.Open(cfg.URLsFile)
		if err != nil {
			fmt.Printf("[-] Error opening URLs file: %v\n", err)
			os.Exit(1)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				urls = append(urls, line)
			}
		}
		file.Close()
		return urls
	}

	if shouldRunDiscovery {
		fmt.Println("\n[STAGE 1] JavaScript Discovery")
		dm := discovery.NewDiscoveryManager(cfg)
		foundURLs := dm.Run()
		return foundURLs
	}

	return urls
}

// runDownloadStage handles downloading logic
func runDownloadStage(cfg *config.Config, urls []string) []*core.JSFile {
	var rawJSFiles []*core.JSFile

	if cfg.RawDir != "" {
		fmt.Printf("\n[STAGE 2] Loading Raw Files from %s (Skipping Download)\n", cfg.RawDir)
		files, err := os.ReadDir(cfg.RawDir)
		if err != nil {
			fmt.Printf("[-] Error reading raw directory: %v\n", err)
			os.Exit(1)
		}

		for _, f := range files {
			if !f.IsDir() {
				rawJSFiles = append(rawJSFiles, &core.JSFile{
					LocalPath:  filepath.Join(cfg.RawDir, f.Name()),
					Filename:   f.Name(),
					Downloaded: true,
				})
			}
		}
		return rawJSFiles
	}

	if len(urls) == 0 {
		return rawJSFiles
	}

	// Prompt after Discovery (moved here to decouple)
	if len(urls) > 0 {
		choice, start, end := ui.PromptDiscoveryChoice(urls)
		if choice == 5 { // Exit
			os.Exit(0)
		}

		urlsToDownload := urls
		if choice == 2 || choice == 3 {
			if end > len(urlsToDownload) {
				end = len(urlsToDownload)
			}
			if start < 0 {
				start = 0
			}
			urlsToDownload = urlsToDownload[start:end]
		} else if choice == 4 {
			urlsToDownload = []string{}
			fmt.Println("[*] Skipping download stage.")
			return rawJSFiles
		}

		// Filter out third-party libraries (GTM, jQuery, etc.)
		if len(urlsToDownload) > 0 {
			originalCount := len(urlsToDownload)
			var filtered []string
			for _, url := range urlsToDownload {
				if !utils.ShouldSkipThirdPartyJS(url) {
					filtered = append(filtered, url)
				}
			}
			skipped := originalCount - len(filtered)
			if skipped > 0 {
				fmt.Printf("\n[*] Filtered out %d third-party libraries (GTM, jQuery, Analytics, etc.)\n", skipped)
				fmt.Printf("[*] %d custom source files remaining for download\n", len(filtered))
			}
			urlsToDownload = filtered
		}

		// Check for existing raw files
		rawDir := filepath.Join(cfg.OutputDir, "js_files", "raw")
		shouldDownload := true

		if _, err := os.Stat(rawDir); err == nil {
			files, _ := os.ReadDir(rawDir)
			jsCount := 0
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".js") {
					jsCount++
				}
			}

			if jsCount > 0 {
				fmt.Printf("\n[!] Found %d existing raw JS files in output directory.\n", jsCount)
				ans := ui.Prompt("    Skip download and use these? (Y/n) [Y]:")
				if ans == "" || strings.ToLower(ans) == "y" {
					shouldDownload = false
					fmt.Println("[*] Loading existing raw files...")
					for _, f := range files {
						if !f.IsDir() && strings.HasSuffix(f.Name(), ".js") {
							rawJSFiles = append(rawJSFiles, &core.JSFile{
								LocalPath:  filepath.Join(rawDir, f.Name()),
								Filename:   f.Name(),
								Downloaded: true,
							})
						}
					}
				}
			}
		}

		if shouldDownload {
			fmt.Println("\n[STAGE 2] JavaScript Download")
			dl := download.NewDownloader(cfg)
			rawJSFiles = dl.Run(urlsToDownload)
		}
	}

	// Check if we have files to proceed
	successCount := 0
	for _, f := range rawJSFiles {
		if f.Downloaded {
			successCount++
		}
	}

	if successCount == 0 {
		fmt.Println("[-] No files to process. Exiting pipeline.")
		os.Exit(0)
	}

	beautifyChoice, bStart, bEnd := ui.PromptDownloadChoice(successCount)

	// Filter files for beautification
	var filesToBeautify []*core.JSFile
	if beautifyChoice != 3 && beautifyChoice != 4 {
		downloadedOnly := make([]*core.JSFile, 0)
		for _, f := range rawJSFiles {
			if f.Downloaded {
				downloadedOnly = append(downloadedOnly, f)
			}
		}

		if bEnd > len(downloadedOnly) {
			bEnd = len(downloadedOnly)
		}
		if bStart < 0 {
			bStart = 0
		}

		if beautifyChoice == 1 || beautifyChoice == 2 {
			filesToBeautify = downloadedOnly[bStart:bEnd]
		}
	}
	// Return processed list
	return filesToBeautify
}

func loadBeautifiedFiles(cfg *config.Config) []string {
	var scanFiles []string
	fmt.Printf("\n[STAGE 3] Loading Beautified Files from %s (Skipping Beautification)\n", cfg.BeautifiedDir)
	files, err := os.ReadDir(cfg.BeautifiedDir)
	if err != nil {
		fmt.Printf("[-] Error reading beautified directory: %v\n", err)
		os.Exit(1)
	}
	for _, f := range files {
		if !f.IsDir() {
			scanFiles = append(scanFiles, filepath.Join(cfg.BeautifiedDir, f.Name()))
		}
	}
	return scanFiles
}

func runBeautifyStage(cfg *config.Config, rawJSFiles []*core.JSFile) []string {
	var scanFiles []string
	if len(rawJSFiles) > 0 {
		shouldBeautify := true
		beautifiedDir := filepath.Join(cfg.OutputDir, "beautified")

		if _, err := os.Stat(beautifiedDir); err == nil {
			files, _ := os.ReadDir(beautifiedDir)
			bCount := 0
			for _, f := range files {
				if !f.IsDir() && strings.HasSuffix(f.Name(), ".js") {
					bCount++
				}
			}

			if bCount > 0 {
				fmt.Printf("\n[!] Found %d existing beautified files.\n", bCount)
				ans := ui.Prompt("    Skip beautification and use these? (Y/n) [Y]:")
				if ans == "" || strings.ToLower(ans) == "y" {
					shouldBeautify = false
					fmt.Println("[*] Loading existing beautified files...")
					for _, f := range files {
						if !f.IsDir() && strings.HasSuffix(f.Name(), ".js") {
							scanFiles = append(scanFiles, filepath.Join(beautifiedDir, f.Name()))
						}
					}
				}
			}
		}

		if shouldBeautify {
			fmt.Println("\n[STAGE 3] Beautification")
			beautifier := beautify.NewBeautifier(cfg)
			beautifiedFiles := beautifier.Run(rawJSFiles)
			scanFiles = beautifiedFiles
		}
	}

	if len(scanFiles) == 0 {
		// Use raw files if beautification skipped
		for _, f := range rawJSFiles {
			if f.Downloaded {
				scanFiles = append(scanFiles, f.LocalPath)
			}
		}
	}
	return scanFiles
}

func runScanStage(cfg *config.Config, state *core.PipelineState, scanChoice int, scanFiles []string) {
	if scanChoice == 1 || scanChoice == 3 {
		fmt.Println("\n[STAGE 4] Secret Scanning")
		ss := scan.NewSecretScanner(cfg)

		// Interactive scan mode selection
		fmt.Println("\n[?] Select Secret Scan Mode:")
		fmt.Println("[1] FAST Scan (Indexed patterns only - Recommended)")
		fmt.Println("    - Uses Aho-Corasick for O(1) matching")
		fmt.Println("    - Skips generic entropy-based checks")
		fmt.Println("[2] DEEP Scan (Include generic fallbacks - Much Slower)")

		mode := ui.Prompt("Select Mode [1-2]")
		if mode == "1" {
			ss.SetSkipGeneric(true)
			ui.Info("Running FAST scan (Skipping generic patterns)")
		} else {
			ss.SetSkipGeneric(false)
			ui.Info("Running DEEP scan (Including generic patterns)")
		}

		state.Secrets = ss.Run(scanFiles)
		fmt.Printf("[+] Found %d secrets\n", len(state.Secrets))

		scan.SaveSecrets(state.Secrets, cfg)
	}

	if scanChoice == 2 || scanChoice == 3 {
		fmt.Println("\n[STAGE 5] Endpoint Extraction")
		es := scan.NewEndpointScanner(cfg)
		state.Endpoints = es.Run(scanFiles)
		fmt.Printf("[+] Found %d endpoints\n", len(state.Endpoints))

		scan.SaveEndpoints(state.Endpoints, cfg)
	}
}

func uniqueAPI(in []string) []string {
	m := make(map[string]bool)
	var out []string
	for _, s := range in {
		if !m[s] {
			m[s] = true
			out = append(out, s)
		}
	}
	return out
}

func printBanner() {
	ui.Println(ui.Bold+ui.Cyan, `
    __ __ ________  _____    _   _____ 
   / //_// ____/\ \/ /   |  / | / /   |
  / ,<  / __/    \  / /| | /  |/ / /| |
 / /| |/ /___    / / ___ |/ /|  / ___ |
/_/ |_/_____/   /_/_/  |_/_/ |_/_/  |_|`)
	ui.Println(ui.Bold+ui.Yellow, "    KEYANA", ui.Reset, "- ", ui.Green, "If it's in JS, KEYANA will find it.")
	ui.Println(ui.Bold+ui.Blue, "    Author:", ui.Reset, " github.com/shaniidev")
	ui.Println(ui.Gray, "    Version: 1.0.0")
	fmt.Println()
}

func loadTemplatePatterns() {
	start := time.Now()
	patterns, err := scan.LoadPatterns()
	if err != nil {
		ui.Warning("Failed to load secret detection patterns: %v", err)
		ui.Warning("Falling back to generic regex scanning only (slower)")
		return
	}
	if len(patterns) == 0 {
		ui.Warning("No secret detection patterns loaded")
		return
	}
	// Set the patterns so the scanner can use them
	scan.SetTemplatePatterns(patterns)
	duration := time.Since(start)
	ui.Success("Loaded %d secret detection patterns in %v", len(patterns), duration)
}

func checkDependencies() {
	requiredTools := []string{"katana", "gau", "waybackurls", "js-beautify", "gitleaks", "trufflehog", "jsluice", "linkfinder"}
	missing := []string{}

	for _, tool := range requiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		ui.Warning("The following optional tools were not found in PATH:")
		for _, m := range missing {
			fmt.Printf("  - %s\n", m)
		}
		ui.Warning("Keyana will skip steps relying on these tools but continue scanning.")
		time.Sleep(2 * time.Second) // Brief pause to let user see warning
	}
}
