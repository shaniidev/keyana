package config

import (
	"flag"
	"path/filepath"
	"strings"
)

type Config struct {
	Domain        string
	ListFile      string
	Concurrency   int
	Timeout       int
	Silent        bool
	Resume        bool
	SkipGeneric   bool // Skip generic fallback patterns (no keywords)
	OutputDir     string
	URLsFile      string
	RawDir        string
	BeautifiedDir string
}

func NewConfig() *Config {
	return &Config{
		Concurrency: 20,
		Timeout:     10,
		OutputDir:   "keyana_output",
	}
}

func (c *Config) ParseFlags() {
	flag.StringVar(&c.Domain, "d", "", "Target domain or URL")
	flag.StringVar(&c.ListFile, "l", "", "List of domains (file)")
	flag.IntVar(&c.Concurrency, "c", 20, "Concurrency level")
	flag.IntVar(&c.Timeout, "timeout", 10, "Timeout in seconds")
	flag.BoolVar(&c.Silent, "silent", false, "Silent mode (no banner)")
	flag.BoolVar(&c.Resume, "resume", false, "Resume previous session")
	flag.StringVar(&c.OutputDir, "o", "keyana_output", "Output directory")

	// Modular Execution Flags
	flag.StringVar(&c.URLsFile, "urls", "", "File containing URLs to download (Skips Discovery)")
	flag.StringVar(&c.RawDir, "raw", "", "Directory containing raw JS files (Skips Discovery & Download)")
	flag.StringVar(&c.BeautifiedDir, "beautified", "", "Directory containing beautified JS files (Skips all previous stages, goes to Scan)")

	flag.Parse()

	if c.Domain == "" && c.ListFile == "" {
		// handle usage or error
	}

	// Ensure absolute path for output dir
	absPath, err := filepath.Abs(c.OutputDir)
	if err == nil {
		c.OutputDir = absPath
	}

	// Append domain to output directory
	if c.Domain != "" {
		safeDomain := c.Domain
		safeDomain = strings.ReplaceAll(safeDomain, "http://", "")
		safeDomain = strings.ReplaceAll(safeDomain, "https://", "")
		safeDomain = strings.ReplaceAll(safeDomain, "/", "_")
		safeDomain = strings.ReplaceAll(safeDomain, ":", "_")
		c.OutputDir = filepath.Join(c.OutputDir, safeDomain)
	}
}
