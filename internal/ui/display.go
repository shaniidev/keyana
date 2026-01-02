package ui

import (
	"fmt"
	"strings"
	"sync"
)

// ProgressBar simple progress indicator
type ProgressBar struct {
	Total   int
	Current int
	Width   int
	Prefix  string
	Mu      sync.Mutex
}

func NewProgressBar(total int, prefix string) *ProgressBar {
	return &ProgressBar{
		Total:  total,
		Width:  40,
		Prefix: prefix,
	}
}

func (p *ProgressBar) Increment() {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	p.Current++
	p.render()
}

func (p *ProgressBar) Add(n int) {
	p.Mu.Lock()
	defer p.Mu.Unlock()
	p.Current += n
	p.render()
}

func (p *ProgressBar) render() {
	percent := float64(p.Current) / float64(p.Total)
	if percent > 1.0 {
		percent = 1.0
	}

	filled := int(float64(p.Width) * percent)
	if filled > p.Width {
		filled = p.Width
	}
	if filled < 0 {
		filled = 0
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", p.Width-filled)

	// Fix format string: %.1f%% for percentage, %d/%d for counts
	fmt.Printf("\r%s %s [%.1f%%] (%d/%d)   ",
		Blue+p.Prefix+Reset,
		Cyan+bar+Reset,
		percent*100,
		p.Current, p.Total,
	)

	if p.Current >= p.Total {
		fmt.Println() // New line on completion
	}
}

// PrintTable prints a list of items in a simple table format
// Limits display to 'limit' items
func PrintTable(items []string, title string, limit int) {
	Section(title)

	// Header
	fmt.Printf("%s\n", strings.Repeat("-", 60))
	fmt.Printf("%-5s | %s\n", "#", "URL")
	fmt.Printf("%s\n", strings.Repeat("-", 60))

	count := len(items)
	displayCount := count
	if displayCount > limit {
		displayCount = limit
	}

	for i := 0; i < displayCount; i++ {
		url := items[i]
		// Truncate URL if too long
		if len(url) > 50 {
			url = "..." + url[len(url)-47:]
		}
		// Print count in default/white, URL in Green
		fmt.Printf("%-5d | %s%s%s\n", i+1, Green, url, Reset)
	}

	if count > limit {
		fmt.Printf("%s\n", strings.Repeat("-", 60))
		fmt.Printf(Yellow+"... and %d more files not shown (displaying first %d)."+Reset+"\n", count-limit, limit)
		fmt.Printf(Green+"All %d files are available for download/processing."+Reset+"\n", count)
	}
	fmt.Printf("%s\n", strings.Repeat("-", 60))
}
