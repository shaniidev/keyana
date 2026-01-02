package scan

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/coregx/coregex"
	"gopkg.in/yaml.v3"
)

//go:embed patterns
var embeddedPatterns embed.FS

// SetTemplatePatterns sets the loaded patterns for use by the scanner
func SetTemplatePatterns(patterns []CompiledPattern) {
	templatePatterns = patterns
	templatePatternsLoaded = true
	// Build the Aho-Corasick engine for rapid scanning
	BuildEngine(patterns)
}

// GetPatternCount returns the number of loaded patterns
func GetPatternCount() int {
	return len(templatePatterns)
}

type PatternTemplate struct {
	ID              string   `yaml:"id"`
	Name            string   `yaml:"name"`
	Description     string   `yaml:"description"`
	Regex           string   `yaml:"regex"`
	Confidence      int      `yaml:"confidence"`
	Severity        string   `yaml:"severity"`
	EntropyCheck    bool     `yaml:"entropy_check"`
	MinEntropy      float64  `yaml:"min_entropy"`
	MinLength       int      `yaml:"min_length"`
	MaxLength       int      `yaml:"max_length"`
	Tags            []string `yaml:"tags"`
	ContextKeywords []string `yaml:"context_keywords"`
	References      []string `yaml:"references"`
}

type PatternFile struct {
	Name     string            `yaml:"name"`
	Version  string            `yaml:"version"`
	Author   string            `yaml:"author"`
	Category string            `yaml:"category"`
	Provider string            `yaml:"provider"`
	Patterns []PatternTemplate `yaml:"patterns"`
}

type CompiledPattern struct {
	ID           string
	Name         string
	Regex        *coregex.Regexp
	RegexString  string // Store original regex for optimization
	Confidence   int
	Severity     string
	EntropyCheck bool
	MinEntropy   float64
	Tags         []string
	Mutex        *sync.Mutex // Protects Regex from concurrent access (lazy DFA)
}

type PatternLoader struct {
	patterns []CompiledPattern
	mu       sync.RWMutex
	loaded   bool
}

var globalLoader = &PatternLoader{}

func LoadPatterns() ([]CompiledPattern, error) {
	globalLoader.mu.Lock()
	defer globalLoader.mu.Unlock()

	if globalLoader.loaded {
		return globalLoader.patterns, nil
	}

	var allPatterns []CompiledPattern
	var loadErrors []string

	err := fs.WalkDir(embeddedPatterns, "patterns", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}

		data, err := embeddedPatterns.ReadFile(path)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: read error: %v", path, err))
			return nil
		}

		var file PatternFile
		if err := yaml.Unmarshal(data, &file); err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: parse error: %v", path, err))
			return nil
		}

		for _, pt := range file.Patterns {
			compiled, err := compilePattern(pt)
			if err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("%s/%s: %v", filepath.Base(path), pt.ID, err))
				continue
			}
			allPatterns = append(allPatterns, compiled)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk patterns directory: %w", err)
	}

	globalLoader.patterns = allPatterns
	globalLoader.loaded = true

	if len(loadErrors) > 0 && len(allPatterns) == 0 {
		return nil, fmt.Errorf("no patterns loaded, errors: %v", loadErrors)
	}

	return allPatterns, nil
}

func compilePattern(pt PatternTemplate) (CompiledPattern, error) {
	re, err := coregex.Compile(pt.Regex)
	if err != nil {
		return CompiledPattern{}, fmt.Errorf("invalid regex: %w", err)
	}

	severity := pt.Severity
	if severity == "" {
		severity = "medium"
	}

	confidence := pt.Confidence
	if confidence == 0 {
		confidence = 75
	}

	return CompiledPattern{
		ID:           pt.ID,
		Name:         pt.Name,
		Regex:        re,
		RegexString:  pt.Regex, // Store it
		Confidence:   confidence,
		Severity:     severity,
		EntropyCheck: pt.EntropyCheck,
		MinEntropy:   pt.MinEntropy,
		Tags:         pt.Tags,
		Mutex:        &sync.Mutex{}, // Initialize mutex
	}, nil
}

func GetPatterns() []CompiledPattern {
	globalLoader.mu.RLock()
	if globalLoader.loaded {
		defer globalLoader.mu.RUnlock()
		return globalLoader.patterns
	}
	globalLoader.mu.RUnlock()

	patterns, err := LoadPatterns()
	if err != nil {
		return nil
	}
	return patterns
}

func PatternCount() int {
	return len(GetPatterns())
}

func GetPatternsByTag(tag string) []CompiledPattern {
	var result []CompiledPattern
	for _, p := range GetPatterns() {
		for _, t := range p.Tags {
			if strings.EqualFold(t, tag) {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

func GetPatternsBySeverity(severity string) []CompiledPattern {
	var result []CompiledPattern
	for _, p := range GetPatterns() {
		if strings.EqualFold(p.Severity, severity) {
			result = append(result, p)
		}
	}
	return result
}
