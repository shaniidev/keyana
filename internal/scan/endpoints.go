package scan

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"keyana/internal/config"
	"keyana/internal/core"
	"keyana/internal/ui"
)

type EndpointScanner struct {
	Config *config.Config
}

func NewEndpointScanner(cfg *config.Config) *EndpointScanner {
	return &EndpointScanner{Config: cfg}
}

func (e *EndpointScanner) Run(files []string) []core.Endpoint {
	var endpoints []core.Endpoint
	var mu sync.Mutex

	// Create log file (Append Mode)
	logPath := filepath.Join(e.Config.OutputDir, "logs", "endpoints_scan.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	// Log header
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "KEYANA - ENDPOINT SCANNING LOG\n")
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "Started: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Total files to scan: %d\n\n", len(files))

	// 1. Regex
	ui.Info("Starting Regex Endpoint Extraction...")
	fmt.Fprintf(logFile, "--- REGEX SCANNER ---\n")
	startTime := time.Now()
	bar := ui.NewProgressBar(len(files), "Regex Endpoints")
	res := e.runRegexScan(files, bar)
	endpoints = append(endpoints, res...)
	fmt.Fprintf(logFile, "Completed in: %s\n", time.Since(startTime))
	fmt.Fprintf(logFile, "Findings: %d\n\n", len(res))

	// 2. LinkFinder
	ui.Info("Running LinkFinder...")
	fmt.Fprintf(logFile, "--- LINKFINDER SCANNER ---\n")
	startTime = time.Now()
	fmt.Fprintf(logFile, "Command: linkfinder -i <file> -o cli\n")
	barLF := ui.NewProgressBar(len(files), "LinkFinder")

	// Run parallel on files
	sem := make(chan struct{}, e.Config.Concurrency)
	var wg sync.WaitGroup
	var logMu sync.Mutex // Mutex for thread-safe log writes
	var linkFinderResults []core.Endpoint
	var linkFinderErrors int

	for _, f := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			defer barLF.Increment()

			res, err := e.runLinkFinder(filePath, logFile, &logMu)
			mu.Lock()
			if err != nil {
				linkFinderErrors++
			} else {
				linkFinderResults = append(linkFinderResults, res...)
				endpoints = append(endpoints, res...)
			}
			mu.Unlock()
		}(f)
	}
	wg.Wait()
	fmt.Fprintf(logFile, "Completed in: %s\n", time.Since(startTime))
	fmt.Fprintf(logFile, "Findings: %d\n", len(linkFinderResults))
	if linkFinderErrors > 0 {
		fmt.Fprintf(logFile, "Errors: %d files had errors (see above for details)\n", linkFinderErrors)
	}
	if len(linkFinderResults) == 0 && linkFinderErrors == 0 {
		fmt.Fprintf(logFile, "Status: No findings (scanner executed successfully)\n")
	}
	fmt.Fprintf(logFile, "\n")

	// Summary
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "SUMMARY\n")
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "Total endpoints found: %d\n", len(endpoints))
	fmt.Fprintf(logFile, "Completed: %s\n", time.Now().Format(time.RFC3339))

	fmt.Printf("[+] Endpoints scan log saved: %s\n", logPath)

	return endpoints
}

func (e *EndpointScanner) runRegexScan(files []string, bar *ui.ProgressBar) []core.Endpoint {
	var endpoints []core.Endpoint

	// Pattern 1: Absolute paths
	absolutePathRe := regexp.MustCompile(`(?:\"|'|` + "`" + `)(/[a-zA-Z0-9_/-]{3,}[a-zA-Z0-9_/])(?:\"|'|` + "`" + `)`)

	// Pattern 2: API paths
	apiPathRe := regexp.MustCompile(`(?:\"|'|` + "`" + `)(/(api|v\d+|graphql|rest|endpoint|service|query|mutation|webhook|callback)[a-zA-Z0-9_/-]*[a-zA-Z0-9_/])(?:\"|'|` + "`" + `)`)

	// Pattern 3: Server-side files
	fileEndpointRe := regexp.MustCompile(`(?:\"|'|` + "`" + `)([a-zA-Z0-9_/-]+\.(php|asp|aspx|jsp|do|action|cgi|pl|py))(?:\"|'|` + "`" + `)`)

	// False positive patterns to exclude
	excludePatterns := []*regexp.Regexp{
		// Module imports and relative paths
		regexp.MustCompile(`^\./[a-z-]+$`),
		regexp.MustCompile(`^\.\./[a-z-]+$`),

		// CSS/jQuery selectors
		regexp.MustCompile(`^[a-z]+\.[a-z]+$`),
		regexp.MustCompile(`(?i)^(click|hover|focus|blur|change|submit|keydown|keyup|mouseenter|mouseleave|tap)\.`),

		// npm/static library paths
		regexp.MustCompile(`/static-\d+\.\d+`),
		regexp.MustCompile(`^/[a-z-]+/static-`),

		// Common false positives
		regexp.MustCompile(`^/(a|store|draft)$`),
		regexp.MustCompile(`\.(css|js|png|jpg|jpeg|gif|svg|woff|woff2|ttf|eot|webp|ico)$`),

		// React/framework specific
		regexp.MustCompile(`(?i)^(react|redux|moment)\.`),
		regexp.MustCompile(`^/content-|^/browserslist-|^/head-|^/hub-`),
	}

	for _, f := range files {
		bar.Increment()
		content, err := os.ReadFile(f)
		if err != nil {
			continue
		}

		contentStr := string(content)
		seen := make(map[string]bool)

		// Process all three patterns
		patterns := []*regexp.Regexp{absolutePathRe, apiPathRe, fileEndpointRe}
		for _, re := range patterns {
			matches := re.FindAllStringSubmatch(contentStr, -1)
			for _, m := range matches {
				if len(m) < 2 {
					continue
				}

				cleanPath := m[1]

				// Length filters
				if len(cleanPath) < 4 || len(cleanPath) > 200 {
					continue
				}

				// Post-match filtering
				if strings.HasPrefix(cleanPath, "//") {
					continue
				}

				if strings.HasPrefix(cleanPath, "/static-") ||
					strings.HasPrefix(cleanPath, "/fonts/") ||
					strings.HasPrefix(cleanPath, "/assets/") {
					continue
				}

				// Check against exclude patterns
				shouldExclude := false
				for _, excludeRe := range excludePatterns {
					if excludeRe.MatchString(cleanPath) {
						shouldExclude = true
						break
					}
				}
				if shouldExclude {
					continue
				}

				// Additional filters
				lowerPath := strings.ToLower(cleanPath)

				if strings.Contains(lowerPath, "static-") ||
					strings.Contains(lowerPath, "node_modules") ||
					strings.Count(cleanPath, "/") > 10 {
					continue
				}

				if seen[cleanPath] {
					continue
				}
				seen[cleanPath] = true

				endpoints = append(endpoints, core.Endpoint{
					Path:   cleanPath,
					Method: "GET",
					File:   f,
					Source: "Regex",
				})
			}
		}
	}
	return endpoints
}

func (e *EndpointScanner) runLinkFinder(filePath string, logFile *os.File, logMu *sync.Mutex) ([]core.Endpoint, error) {
	cmd := exec.Command("linkfinder", "-i", filePath, "-o", "cli")

	// Capture both stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logMu.Lock()
		fmt.Fprintf(logFile, "[ERROR] File: %s - %v\n", filePath, err)
		if stderr.Len() > 0 {
			fmt.Fprintf(logFile, "  Stderr: %s\n", stderr.String())
		}
		logMu.Unlock()
		return nil, err
	}

	var results []core.Endpoint
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Running") {
			continue
		}

		results = append(results, core.Endpoint{
			Path:   line,
			Method: "GET",
			File:   filePath,
			Source: "LinkFinder",
		})
	}

	return results, nil
}
