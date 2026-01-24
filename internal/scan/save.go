package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/shaniidev/keyana/internal/config"
	"github.com/shaniidev/keyana/internal/core"
)

func SaveSecrets(secrets []core.Secret, cfg *config.Config) {
	// Group secrets by detector
	detectorGroups := make(map[string][]core.Secret)
	for _, s := range secrets {
		detectorGroups[s.Detector] = append(detectorGroups[s.Detector], s)
	}

	var sb strings.Builder
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("KEYANA - SECRET DETECTION REPORT\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")
	sb.WriteString(fmt.Sprintf("Total Secrets Found: %d\n\n", len(secrets)))

	// Define detector order for consistent output (all sections shown)
	detectorOrder := []string{
		"Template (critical)",
		"Template (high)",
		"Template (medium)",
		"Regex (Vendor)",
		"Regex (Entropy)",
		"Entropy (Pure)",
		"gitleaks",
		"jsluice",
		"trufflehog",
	}

	for _, detector := range detectorOrder {
		findings, exists := detectorGroups[detector]
		count := 0
		if exists {
			count = len(findings)
		}

		// Section header (always shown)
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString(fmt.Sprintf("DETECTOR: %s (%d findings)\n", strings.ToUpper(detector), count))
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		if count == 0 {
			sb.WriteString("  No findings from this detector.\n\n")
			continue
		}

		// List findings
		for i, s := range findings {
			fmt.Fprintf(&sb, "[Finding #%d]\n", i+1)
			fmt.Fprintf(&sb, "  File: %s\n", s.File)
			fmt.Fprintf(&sb, "  Type: %s\n", s.Type)
			fmt.Fprintf(&sb, "  Line: %d\n", s.Line)
			fmt.Fprintf(&sb, "  Secret: %s\n", s.Value)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	outputPath := filepath.Join(cfg.OutputDir, "reports", "secrets.txt")
	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[-] Error opening secrets report: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(sb.String()); err != nil {
		fmt.Printf("[-] Error saving secrets: %v\n", err)
	} else {
		fmt.Printf("[+] Secrets Report Appended: %s\n", outputPath)
	}
}

func SaveEndpoints(endpoints []core.Endpoint, cfg *config.Config) {
	// Group endpoints by source
	sourceGroups := make(map[string][]core.Endpoint)
	for _, e := range endpoints {
		sourceGroups[e.Source] = append(sourceGroups[e.Source], e)
	}

	var sb strings.Builder
	sb.WriteString(strings.Repeat("=", 80) + "\n")
	sb.WriteString("KEYANA - API ENDPOINTS REPORT\n")
	sb.WriteString(strings.Repeat("=", 80) + "\n\n")

	sb.WriteString(fmt.Sprintf("Total endpoints found: %d\n\n", len(endpoints)))

	// Define source order
	sourceOrder := []string{"Regex", "LinkFinder"}

	for _, source := range sourceOrder {
		findings, exists := sourceGroups[source]
		if !exists || len(findings) == 0 {
			continue
		}

		// Section header
		sb.WriteString(strings.Repeat("-", 80) + "\n")
		sb.WriteString(fmt.Sprintf("SOURCE: %s (%d endpoints)\n", strings.ToUpper(source), len(findings)))
		sb.WriteString(strings.Repeat("-", 80) + "\n\n")

		// Deduplicate paths within this source
		uniquePaths := make(map[string]bool)
		for _, e := range findings {
			if !uniquePaths[e.Path] {
				uniquePaths[e.Path] = true
				fmt.Fprintf(&sb, "%s\n", e.Path)
			}
		}
		sb.WriteString("\n")
	}

	outputPath := filepath.Join(cfg.OutputDir, "reports", "endpoints.txt")
	f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[-] Error opening endpoints report: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(sb.String()); err != nil {
		fmt.Printf("[-] Error saving endpoints: %v\n", err)
	} else {
		fmt.Printf("[+] Endpoints Report Appended: %s\n", outputPath)
	}
}

// SaveScanLog saves detailed scan logs to logs folder
func SaveScanLog(logName string, content string, cfg *config.Config) {
	logPath := filepath.Join(cfg.OutputDir, "logs", logName)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("[-] Error opening log %s: %v\n", logName, err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		fmt.Printf("[-] Error saving log %s: %v\n", logName, err)
	}
}
