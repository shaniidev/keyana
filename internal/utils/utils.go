package utils

import (
	"fmt"
	"math"
	"net/url"
	"os"
	"strings"
)

// CalculateEntropy computes the Shannon entropy of a string
func CalculateEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	// Count character frequencies
	freqs := make(map[rune]float64)
	for _, r := range s {
		freqs[r]++
	}

	// Calculate entropy
	var entropy float64
	total := float64(len(s))
	for _, count := range freqs {
		p := count / total
		entropy -= p * math.Log2(p)
	}

	return entropy
}

// IsJSFile checks if the URL points to a JS file
func IsJSFile(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		// If parse fails, simple string check
		return strings.Contains(rawURL, ".js")
	}

	path := u.Path
	return strings.HasSuffix(strings.ToLower(path), ".js")
}

// ShouldSkipThirdPartyJS checks if a JS URL is a common third-party library
// Returns true if the file should be skipped (not downloaded)
func ShouldSkipThirdPartyJS(rawURL string) bool {
	lowerURL := strings.ToLower(rawURL)

	// Skip node_modules folder entirely
	if strings.Contains(lowerURL, "node_modules") {
		return true
	}

	// Common third-party CDNs - skip entirely
	cdnDomains := []string{
		"googleapis.com",
		"gstatic.com",
		"google-analytics.com",
		"googletagmanager.com",
		"doubleclick.net",
		"facebook.net",
		"connect.facebook.net",
		"cloudflare.com",
		"cdnjs.cloudflare.com",
		"ajax.cloudflare.com",
		"cdn.jsdelivr.net",
		"unpkg.com",
		"code.jquery.com",
		"ajax.googleapis.com",
		"maxcdn.bootstrapcdn.com",
		"stackpath.bootstrapcdn.com",
		"cdn.bootcss.com",
		"ajax.aspnetcdn.com",
		"microsoft.com/ajax",
	}

	for _, cdn := range cdnDomains {
		if strings.Contains(lowerURL, cdn) {
			return true
		}
	}

	// Common library filenames - skip these specific files
	skipFilenames := []string{
		// Analytics & Tracking
		"gtm.js",
		"gtag.js",
		"ga.js",
		"analytics.js",
		"google-analytics",
		"googletagmanager",
		"facebook-pixel",
		"fbevents.js",
		"pixel.js",

		// jQuery (all versions)
		"jquery.js",
		"jquery.min.js",
		"jquery-",
		"jquery.",

		// Bootstrap
		"bootstrap.js",
		"bootstrap.min.js",
		"bootstrap.bundle",

		// Other common libraries
		"modernizr",
		"polyfill",
		"shim.js",
		"html5shiv",
		"respond.min.js",
		"selectivizr",

		// Fonts & Icon libraries
		"fontawesome",
		"font-awesome",

		// Social media widgets
		"twitter.com/widgets",
		"platform.twitter",
		"linkedin.com/embed",
		"instagram.com/embed",

		// Ad networks
		"doubleclick",
		"googlesyndication",
		"adservice",
		"ads.js",

		// Monitoring/Error tracking
		"sentry",
		"newrelic",
		"hotjar",
		"clarity.ms",
	}

	for _, skip := range skipFilenames {
		if strings.Contains(lowerURL, skip) {
			return true
		}
	}

	return false
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteLines writes a slice of strings to a file
func WriteLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(f, line); err != nil {
			return err
		}
	}
	return nil
}
