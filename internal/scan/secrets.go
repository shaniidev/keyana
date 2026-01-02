package scan

import (
	"bufio"
	"bytes"
	"encoding/json"
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
	"keyana/internal/utils"
)

// templatePatternsLoaded tracks if templates were loaded
var templatePatternsLoaded bool
var templatePatterns []CompiledPattern
var templateLoadOnce sync.Once

// ============================================================================
// PACKAGE-LEVEL COMPILED REGEX (Performance Optimization)
// ============================================================================

// Generic pattern for variable assignments with secret keywords
var genericSecretRe = regexp.MustCompile(`(?i)\b((?:api|token|secret|password|auth|access|private|bearer|credential)(?:_?key|_?token|_?secret|_?pass|_?pwd|_?id)?)\s*[:=]\s*['"]([a-zA-Z0-9_\-+/=]{12,})['"]`)

// Patterns are loaded from YAML templates
// See template_loader.go and patterns/*.yaml files

// False positive patterns to exclude (compiled once)
var falsePositivePatterns = []*regexp.Regexp{
	// UI/UX terms
	regexp.MustCompile(`(?i)\b(invalid|error|message|description|title|label|placeholder|text|name|type|class|style|attribute|hint|warning|info)\b`),
	regexp.MustCompile(`(?i)\b(dialog|modal|button|input|form|field|element|component|container|wrapper|panel|view|screen)\b`),

	// Function naming conventions
	regexp.MustCompile(`(?i)^(get|set|on|is|has|did|will|should|can|handle|render|update|create|delete|fetch|load|save|validate|check|parse)[A-Z]`),

	// CSS/Style patterns
	regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+`),
	regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|rgb|rgba|hsl)$`),

	// Framework prefixes
	regexp.MustCompile(`(?i)^(clip|static|mdl|react|redux|form|wizard|tabpanel|mui|ant|chakra|next)`),

	// Common test/placeholder values
	regexp.MustCompile(`(?i)(example|test|demo|sample|placeholder|dummy|fake|mock|lorem|ipsum)`),
	regexp.MustCompile(`(?i)^(xxx+|___+|\.\.\.+|null|undefined|none|empty)$`),

	// Hash/ID patterns that aren't secrets
	regexp.MustCompile(`^[0-9a-f]{32}$`), // MD5-like (without context)
	regexp.MustCompile(`^[0-9a-f]{40}$`), // SHA1-like (git commit)
	regexp.MustCompile(`^[0-9a-f]{64}$`), // SHA256-like

	// Base64/Base58 alphabet definitions (NOT secrets)
	regexp.MustCompile(`^[A-Za-z0-9+/=]{64,}$`),                                           // Base64 alphabet
	regexp.MustCompile(`^[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+$`), // Base58 alphabet
	regexp.MustCompile(`^[A-Z]{52,}$`),                                                    // All caps alphabet (font data etc)

	// Unicode/Font data
	regexp.MustCompile(`^[A-Za-z0-9/+]{500,}$`), // Long base64 encoded data (fonts, images)
}

// High-entropy string literal pattern
var stringLiteralRe = regexp.MustCompile(`['"]([a-zA-Z0-9_\-+/=]{32,})['"]`)

// ============================================================================
// SCANNER IMPLEMENTATION
// ============================================================================

type SecretScanner struct {
	Config *config.Config
}

func NewSecretScanner(cfg *config.Config) *SecretScanner {
	return &SecretScanner{Config: cfg}
}

func (s *SecretScanner) SetSkipGeneric(skip bool) {
	s.Config.SkipGeneric = skip
}

func (s *SecretScanner) Run(files []string) []core.Secret {
	var secrets []core.Secret

	// Create log file (Append Mode)
	logPath := filepath.Join(s.Config.OutputDir, "logs", "secrets_scan.log")
	os.MkdirAll(filepath.Dir(logPath), 0755)
	logFile, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer logFile.Close()

	// Log header
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "KEYANA - SECRET SCANNING LOG\n")
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "Started: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(logFile, "Total files to scan: %d\n\n", len(files))

	// 1. Keyana Scan (Templates + Heuristics)
	ui.Info("Starting Keyana Secret Scanner...")
	fmt.Fprintf(logFile, "--- KEYANA  SECRET SCANNER ---\n")
	startTime := time.Now()
	bar := ui.NewProgressBar(len(files), "Keyana Scan")
	res := s.runRegexScan(files, bar)
	secrets = append(secrets, res...)
	fmt.Fprintf(logFile, "Completed in: %s\n", time.Since(startTime))
	fmt.Fprintf(logFile, "Findings: %d\n\n", len(res))

	// 2. Gitleaks
	ui.Info("Starting Gitleaks...")
	fmt.Fprintf(logFile, "--- GITLEAKS SCANNER ---\n")
	startTime = time.Now()
	resGL := s.runGitleaks(logFile)
	secrets = append(secrets, resGL...)
	fmt.Fprintf(logFile, "Completed in: %s\n", time.Since(startTime))
	if len(resGL) == 0 {
		fmt.Fprintf(logFile, "Status: No findings\n")
	}
	fmt.Fprintf(logFile, "Findings: %d\n\n", len(resGL))
	ui.Success("Gitleaks finished")

	// 3. JSLuice
	ui.Info("Starting JSLuice...")
	fmt.Fprintf(logFile, "--- JSLUICE SCANNER ---\n")
	startTime = time.Now()
	fmt.Fprintf(logFile, "Command: jsluice secrets <file>\n")
	barJS := ui.NewProgressBar(len(files), "JSLuice")

	var mu sync.Mutex
	var wg sync.WaitGroup
	var logMu sync.Mutex
	sem := make(chan struct{}, s.Config.Concurrency)
	var jsLuiceResults []core.Secret
	var jsLuiceErrors int

	for _, f := range files {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			defer barJS.Increment()

			r, err := s.runJSLuice(filePath, logFile, &logMu)
			mu.Lock()
			if err != nil {
				jsLuiceErrors++
			} else {
				jsLuiceResults = append(jsLuiceResults, r...)
				secrets = append(secrets, r...)
			}
			mu.Unlock()
		}(f)
	}
	wg.Wait()
	fmt.Fprintf(logFile, "Completed in: %s\n", time.Since(startTime))
	fmt.Fprintf(logFile, "Files scanned: %d\n", len(files))
	fmt.Fprintf(logFile, "Files with errors: %d\n", jsLuiceErrors)
	fmt.Fprintf(logFile, "Findings: %d\n", len(jsLuiceResults))
	if len(jsLuiceResults) == 0 && jsLuiceErrors == 0 {
		fmt.Fprintf(logFile, "Status: No findings (scanner executed successfully)\n")
	}
	fmt.Fprintf(logFile, "\n")

	// 4. TruffleHog
	ui.Info("Starting TruffleHog...")
	fmt.Fprintf(logFile, "--- TRUFFLEHOG SCANNER ---\n")
	startTime = time.Now()
	truffleHogResults := s.runTruffleHog(logFile)
	secrets = append(secrets, truffleHogResults...)
	fmt.Fprintf(logFile, "Completed in: %s\n", time.Since(startTime))
	fmt.Fprintf(logFile, "Findings: %d\n", len(truffleHogResults))
	if len(truffleHogResults) == 0 {
		fmt.Fprintf(logFile, "Status: No findings (scanner executed successfully)\n")
	}
	fmt.Fprintf(logFile, "\n")

	// Summary
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "SUMMARY\n")
	fmt.Fprintf(logFile, "=================================================================\n")
	fmt.Fprintf(logFile, "Total secrets found: %d\n", len(secrets))
	fmt.Fprintf(logFile, "Completed: %s\n", time.Now().Format(time.RFC3339))

	fmt.Printf("[+] Secrets scan log saved: %s\n", logPath)

	return secrets
}

func (s *SecretScanner) runRegexScan(files []string, bar *ui.ProgressBar) []core.Secret {
	numWorkers := 8
	if s.Config.Concurrency > 0 {
		numWorkers = s.Config.Concurrency
	}

	// Channels for parallel processing
	fileChan := make(chan string, len(files))
	resultChan := make(chan []core.Secret, len(files))
	var wg sync.WaitGroup

	// Start worker pool
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for filePath := range fileChan {
				secrets := s.scanSingleFile(filePath)
				resultChan <- secrets
			}
		}()
	}

	// Feed files to workers
	go func() {
		for _, f := range files {
			fileChan <- f
		}
		close(fileChan)
	}()

	// Collect progress
	go func() {
		for range resultChan {
			bar.Increment()
		}
	}()

	wg.Wait()
	close(resultChan)

	// Collect all results
	var found []core.Secret
	seenSecrets := make(map[string]bool)

	// Drain any remaining results
	for secrets := range resultChan {
		for _, sec := range secrets {
			if !seenSecrets[sec.Value] {
				seenSecrets[sec.Value] = true
				found = append(found, sec)
			}
		}
	}

	return found
}

// scanSingleFile processes one file with all optimizations
func (s *SecretScanner) scanSingleFile(filePath string) []core.Secret {
	var found []core.Secret
	seenInFile := make(map[string]bool)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return found
	}

	// Pre-calculate line positions once (O(n) → O(log n) for lookups)
	linePositions := buildLineIndex(content)

	// 1. Template Patterns (High Performance Aho-Corasick Engine)
	highConfidenceFound := false
	if templatePatternsLoaded {
		// Use the O(1) engine scan
		matches := GlobalEngine.ScanContent(content, filePath, seenInFile, linePositions, s.Config.SkipGeneric)

		for _, m := range matches {
			found = append(found, m)
			highConfidenceFound = true
		}
	}

	// 2. Skip entropy scanning if high-confidence secrets found
	if highConfidenceFound {
		return found
	}

	// 3. Generic Heuristic (key=value patterns with entropy) - ONLY if not skipping generic
	if !s.Config.SkipGeneric {
		matches := genericSecretRe.FindAllSubmatchIndex(content, -1)
		for _, mIdx := range matches {
			if len(mIdx) < 6 {
				continue
			}

			varName := string(content[mIdx[2]:mIdx[3]])
			val := string(content[mIdx[4]:mIdx[5]])

			if seenInFile[val] || len(val) < 16 {
				continue
			}
			if isFalsePositiveVar(varName) || isFalsePositive(val) || isRepeatingPattern(val) {
				continue
			}

			threshold := getEntropyThreshold(len(val))
			if utils.CalculateEntropy(val) > threshold {
				seenInFile[val] = true
				found = append(found, core.Secret{
					Type:     fmt.Sprintf("Potential Secret: %s", varName),
					Value:    val,
					File:     filePath,
					Line:     getLineFromIndex(linePositions, mIdx[0]),
					Detector: "Regex (Entropy)",
				})
			}
		}
	}

	// 4. Pure Entropy Scan - ONLY if not skipping generic
	if !s.Config.SkipGeneric {
		matchesLit := stringLiteralRe.FindAllSubmatchIndex(content, -1)
		for _, mIdx := range matchesLit {
			val := string(content[mIdx[2]:mIdx[3]])

			if seenInFile[val] {
				continue
			}
			if isFalsePositive(val) || isRepeatingPattern(val) {
				continue
			}
			if len(val) == 32 || len(val) == 40 || len(val) == 64 {
				continue // Skip hash-like lengths
			}

			if utils.CalculateEntropy(val) > 5.2 {
				seenInFile[val] = true
				found = append(found, core.Secret{
					Type:     "High Entropy String",
					Value:    val,
					File:     filePath,
					Line:     getLineFromIndex(linePositions, mIdx[0]),
					Detector: "Entropy (Pure)",
				})
			}
		}
	}

	return found
}

// buildLineIndex creates an index of line start positions for fast lookup
func buildLineIndex(content []byte) []int {
	positions := []int{0}
	for i, b := range content {
		if b == '\n' {
			positions = append(positions, i+1)
		}
	}
	return positions
}

// getLineFromIndex returns line number using binary search (O(log n))
func getLineFromIndex(positions []int, offset int) int {
	if offset < 0 {
		return 1
	}
	// Binary search for the line
	lo, hi := 0, len(positions)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if positions[mid] <= offset {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo + 1
}

// runTemplatePatterns scans content using coregex-powered template patterns (1000+)

// getEntropyThreshold returns adaptive threshold based on string length
func getEntropyThreshold(length int) float64 {
	switch {
	case length < 20:
		return 5.0 // Very strict for short strings
	case length < 32:
		return 4.5 // Strict
	case length < 48:
		return 4.2 // Medium
	default:
		return 4.0 // Standard for long strings
	}
}

// isFalsePositive checks if a value matches false positive patterns
func isFalsePositive(val string) bool {
	lowerVal := strings.ToLower(val)

	// Quick keyword checks
	if strings.Contains(lowerVal, "example") || strings.Contains(lowerVal, "test") ||
		strings.Contains(lowerVal, "sample") || strings.Contains(lowerVal, "placeholder") ||
		strings.Contains(lowerVal, "dummy") || strings.Contains(lowerVal, "mock") ||
		strings.Contains(lowerVal, "localhost") || strings.Contains(lowerVal, "127.0.0.1") {
		return true
	}

	// Check regex patterns
	for _, fpRe := range falsePositivePatterns {
		if fpRe.MatchString(val) {
			return true
		}
	}

	return false
}

// isFalsePositiveVar checks if a variable name indicates false positive
func isFalsePositiveVar(varName string) bool {
	lowerVar := strings.ToLower(varName)

	// Common non-secret variable patterns
	fpKeywords := []string{"example", "test", "demo", "sample", "placeholder", "dummy",
		"mock", "fake", "invalid", "error", "message", "description", "label", "title"}

	for _, kw := range fpKeywords {
		if strings.Contains(lowerVar, kw) {
			return true
		}
	}

	// Check regex patterns for variable names
	for _, fpRe := range falsePositivePatterns[:3] { // First 3 patterns are for var names
		if fpRe.MatchString(varName) {
			return true
		}
	}

	return false
}

// isRepeatingPattern checks if a string is mostly repeating characters
func isRepeatingPattern(s string) bool {
	if len(s) < 4 {
		return false
	}

	charCounts := make(map[rune]int)
	for _, c := range s {
		charCounts[c]++
	}

	// If any character appears more than 60% of the time, it's repetitive
	for _, count := range charCounts {
		if float64(count)/float64(len(s)) > 0.6 {
			return true
		}
	}

	// Check for sequential patterns (e.g., "abcabc", "123123")
	if len(s) >= 6 {
		half := s[:len(s)/2]
		if strings.Contains(s, half+half) {
			return true
		}
	}

	return false
}

// ============================================================================
// EXTERNAL SCANNER INTEGRATIONS
// ============================================================================

type GitleaksResult struct {
	Description string `json:"Description"`
	Secret      string `json:"Secret"`
	File        string `json:"File"`
	Match       string `json:"Match"`
	StartLine   int    `json:"StartLine"`
}

func (s *SecretScanner) runGitleaks(logFile *os.File) []core.Secret {
	sourceDir := filepath.Join(s.Config.OutputDir, "beautified")
	tmpReport := filepath.Join(s.Config.OutputDir, "gitleaks_report.json")

	cmdStr := fmt.Sprintf("gitleaks detect --source %s --no-git --report-path %s --exit-code 0", sourceDir, tmpReport)
	fmt.Fprintf(logFile, "Command: %s\n", cmdStr)

	cmd := exec.Command("gitleaks", "detect", "--source", sourceDir, "--no-git", "--report-path", tmpReport, "--exit-code", "0")

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(logFile, "Error executing: %v\n", err)
		fmt.Fprintf(logFile, "Stderr: %s\n", stderr.String())
		return []core.Secret{}
	}

	if stderr.Len() > 0 {
		fmt.Fprintf(logFile, "Stderr: %s\n", stderr.String())
	}

	data, err := os.ReadFile(tmpReport)
	if err != nil {
		fmt.Fprintf(logFile, "Error reading report: %v\n", err)
		return []core.Secret{}
	}

	var glRes []GitleaksResult
	if err := json.Unmarshal(data, &glRes); err != nil {
		fmt.Fprintf(logFile, "Error parsing JSON: %v\n", err)
		return []core.Secret{}
	}

	var results []core.Secret
	for _, r := range glRes {
		results = append(results, core.Secret{
			Type:     fmt.Sprintf("Gitleaks: %s", r.Description),
			Value:    r.Secret,
			File:     r.File,
			Line:     r.StartLine,
			Detector: "gitleaks",
		})
	}

	os.Remove(tmpReport)

	return results
}

func (s *SecretScanner) runJSLuice(filePath string, logFile *os.File, logMu *sync.Mutex) ([]core.Secret, error) {
	cmd := exec.Command("jsluice", "secrets", filePath)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		logMu.Lock()
		fmt.Fprintf(logFile, "[ERROR] %s - %v\n", filepath.Base(filePath), err)
		if stderr.Len() > 0 {
			fmt.Fprintf(logFile, "  Stderr: %s\n", strings.TrimSpace(stderr.String()))
		}
		logMu.Unlock()
		return nil, err
	}

	var secrets []core.Secret
	scanner := bufio.NewScanner(&stdout)

	type JSLuiceOut struct {
		Kind string `json:"kind"`
		Data string `json:"data"`
		Line int    `json:"line"`
	}

	for scanner.Scan() {
		var jso JSLuiceOut
		if err := json.Unmarshal(scanner.Bytes(), &jso); err == nil {
			secrets = append(secrets, core.Secret{
				Type:     fmt.Sprintf("JSLuice: %s", jso.Kind),
				Value:    jso.Data,
				File:     filePath,
				Line:     jso.Line,
				Detector: "jsluice",
			})
		}
	}

	return secrets, nil
}

func (s *SecretScanner) runTruffleHog(logFile *os.File) []core.Secret {
	beautifiedDir := filepath.Join(s.Config.OutputDir, "beautified")

	cmdStr := fmt.Sprintf("trufflehog filesystem %s --json --no-update --no-verification", beautifiedDir)
	fmt.Fprintf(logFile, "Command: %s\n", cmdStr)

	cmd := exec.Command("trufflehog", "filesystem", beautifiedDir, "--json", "--no-update", "--no-verification")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(logFile, "Error executing: %v\n", err)
		if stderr.Len() > 0 {
			fmt.Fprintf(logFile, "Stderr: %s\n", stderr.String())
		}
		return []core.Secret{}
	}

	if stderr.Len() > 0 {
		fmt.Fprintf(logFile, "Stderr: %s\n", stderr.String())
	}

	var secrets []core.Secret
	scanner := bufio.NewScanner(&stdout)

	type TruffleHogResult struct {
		DetectorName   string `json:"DetectorName"`
		Raw            string `json:"Raw"`
		SourceMetadata struct {
			Data struct {
				Filesystem struct {
					File string `json:"file"`
					Line int64  `json:"line"`
				} `json:"Filesystem"`
			} `json:"Data"`
		} `json:"SourceMetadata"`
		Verified bool `json:"Verified"`
	}

	for scanner.Scan() {
		var result TruffleHogResult
		if err := json.Unmarshal(scanner.Bytes(), &result); err == nil {
			secrets = append(secrets, core.Secret{
				Type:     fmt.Sprintf("TruffleHog: %s", result.DetectorName),
				Value:    result.Raw,
				File:     result.SourceMetadata.Data.Filesystem.File,
				Line:     int(result.SourceMetadata.Data.Filesystem.Line),
				Detector: "trufflehog",
			})
		}
	}

	return secrets
}
