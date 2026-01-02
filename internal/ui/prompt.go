package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// PromptUserGeneric generic prompt
func Prompt(question string) string {
	fmt.Print(question + " ")
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// PromptDiscoveryChoice asks the user how to proceed after discovery
func PromptDiscoveryChoice(urls []string) (int, int, int) {
	total := len(urls)

	// Show Table
	PrintTable(urls, "Discovered JavaScript Files", 20)

	Success("Total JS Files Discovered: %d", total)
	Printf(Cyan, "[1] Download ALL JS files\n")
	Printf(Cyan, "[2] Download first N JS files\n")
	Printf(Cyan, "[3] Download range (X-Y)\n")
	Printf(Cyan, "[4] Skip download stage\n")
	Printf(Red, "[5] Exit\n")

	choiceStr := Prompt("\nSelect Option [1-5]:")
	choice, _ := strconv.Atoi(choiceStr)

	start, end := 0, total

	switch choice {
	case 1:
		// All
	case 2:
		nStr := Prompt("Enter N:")
		n, _ := strconv.Atoi(nStr)
		if n > total {
			n = total
		}
		end = n
	case 3:
		rangeStr := Prompt("Enter range (e.g. 10-50):")
		parts := strings.Split(rangeStr, "-")
		if len(parts) == 2 {
			s, _ := strconv.Atoi(parts[0])
			e, _ := strconv.Atoi(parts[1])
			if s < 1 {
				s = 1
			}
			if e > total {
				e = total
			}
			start = s - 1 // 0-indexed
			end = e
		}
	case 4:
		// Skip
	case 5:
		os.Exit(0)
	default:
		return 4, 0, 0
	}

	return choice, start, end
}

// PromptDownloadChoice asks user what to beautify
func PromptDownloadChoice(total int) (int, int, int) {
	Success("Successfully downloaded: %d", total)
	Printf(Cyan, "[1] Beautify all\n")
	Printf(Cyan, "[2] Beautify selected range\n")
	Printf(Cyan, "[3] Skip beautification\n")
	Printf(Red, "[4] Exit\n")

	choiceStr := Prompt("\nSelect Option [1-4]:")
	choice, _ := strconv.Atoi(choiceStr)

	start, end := 0, total

	switch choice {
	case 1:
		// All
	case 2:
		rangeStr := Prompt("Enter range (e.g. 10-50):")
		parts := strings.Split(rangeStr, "-")
		if len(parts) == 2 {
			s, _ := strconv.Atoi(parts[0])
			e, _ := strconv.Atoi(parts[1])
			if s < 1 {
				s = 1
			}
			if e > total {
				e = total
			}
			start = s - 1
			end = e
		}
	case 3:
		// Skip
	case 4:
		os.Exit(0)
	}

	return choice, start, end
}

// PromptScanChoice asks user what to scan
func PromptScanChoice(total int) int {
	Success("Total files for scanning: %d", total)
	Printf(Cyan, "[1] Scan for secrets\n")
	Printf(Cyan, "[2] Scan for endpoints\n")
	Printf(Cyan, "[3] Scan for both\n")
	Printf(Red, "[4] Exit\n")

	choiceStr := Prompt("\nSelect Option [1-4]:")
	choice, _ := strconv.Atoi(choiceStr)

	if choice == 4 {
		os.Exit(0)
	}
	return choice
}
