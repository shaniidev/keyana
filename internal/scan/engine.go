package scan

import (
	"bytes"
	"fmt"
	"keyana/internal/core"
	"keyana/internal/utils"
	"sync"

	"github.com/cloudflare/ahocorasick"
)

// Engine is the high-performance scanning engine
type Engine struct {
	Matcher          *ahocorasick.Matcher
	KeywordIndexMap  map[int][]int // Maps keyword_index (from AC) -> list of pattern indices (in AllPatterns)
	FallbackPatterns []int         // Indices of patterns with no keywords (must always run)
	AllPatterns      []CompiledPattern
	IsReady          bool
	mu               sync.RWMutex
}

var GlobalEngine = &Engine{
	KeywordIndexMap: make(map[int][]int),
}

// BuildEngine compiles the Aho-Corasick matcher from the provided patterns
func BuildEngine(patterns []CompiledPattern) {
	GlobalEngine.mu.Lock()
	defer GlobalEngine.mu.Unlock()

	GlobalEngine.AllPatterns = patterns
	GlobalEngine.KeywordIndexMap = make(map[int][]int)
	GlobalEngine.FallbackPatterns = []int{}

	var keywords []string
	// Temporary map to dedup keywords and track their index in the 'keywords' slice
	// Keyword string -> Index in 'keywords' slice
	keywordToSliceIdx := make(map[string]int)

	for i, p := range patterns {
		kw := ExtractKeyword(p.RegexString)

		if IsValidKeyword(kw) {
			sliceIdx, exists := keywordToSliceIdx[kw]
			if !exists {
				sliceIdx = len(keywords)
				keywords = append(keywords, kw)
				keywordToSliceIdx[kw] = sliceIdx
			}
			GlobalEngine.KeywordIndexMap[sliceIdx] = append(GlobalEngine.KeywordIndexMap[sliceIdx], i)
		} else {
			GlobalEngine.FallbackPatterns = append(GlobalEngine.FallbackPatterns, i)
		}
	}

	// Cloudflare NewStringMatcher takes a slice of strings
	GlobalEngine.Matcher = ahocorasick.NewStringMatcher(keywords)
	GlobalEngine.IsReady = true

	fmt.Printf("[+] Engine optimized: %d patterns indexed via %d unique keywords, %d fallbacks\n",
		len(patterns)-len(GlobalEngine.FallbackPatterns), len(keywords), len(GlobalEngine.FallbackPatterns))
}

// ScanContent runs the high-performance scan on a file
func (e *Engine) ScanContent(content []byte, filePath string, seen map[string]bool, linePositions []int, skipGeneric bool) []core.Secret {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.IsReady {
		return nil
	}

	var found []core.Secret

	// 1. Run Fallback Patterns (Sequential) - ONLY if not skipping generic
	if !skipGeneric {
		for _, idx := range e.FallbackPatterns {
			found = append(found, runSinglePattern(e.AllPatterns[idx], content, filePath, seen, linePositions)...)
		}
	}

	// 2. Run Pre-Filter (Aho-Corasick) - O(n) scan
	// Match returns indices of the keywords found in the content
	matches := e.Matcher.Match(content)

	// 3. Process Hits
	// Collect potential patterns to check
	patternsToCheck := make(map[int]bool)

	for _, matchIdx := range matches {
		if patternIndices, ok := e.KeywordIndexMap[matchIdx]; ok {
			for _, pIdx := range patternIndices {
				patternsToCheck[pIdx] = true
			}
		}
	}

	// 4. Run Triggered Patterns
	for pIdx := range patternsToCheck {
		found = append(found, runSinglePattern(e.AllPatterns[pIdx], content, filePath, seen, linePositions)...)
	}

	return found
}

func runSinglePattern(pattern CompiledPattern, content []byte, filePath string, seen map[string]bool, linePositions []int) []core.Secret {
	var results []core.Secret

	// Coregex lazy DFA is not thread-safe, so we must lock per pattern
	if pattern.Mutex != nil {
		pattern.Mutex.Lock()
		defer pattern.Mutex.Unlock()
	}

	matches := pattern.Regex.FindAll(content, -1)

	for _, match := range matches {
		matchStr := string(match)
		if seen[matchStr] {
			continue
		}

		// Re-implement filters
		if isFalsePositive(matchStr) {
			continue
		}
		if pattern.EntropyCheck && pattern.MinEntropy > 0 {
			if utils.CalculateEntropy(matchStr) < pattern.MinEntropy {
				continue
			}
		}

		idx := bytes.Index(content, match)
		lineNum := getLineFromIndex(linePositions, idx)

		seen[matchStr] = true
		results = append(results, core.Secret{
			Type:     pattern.Name,
			Value:    matchStr,
			File:     filePath,
			Line:     lineNum,
			Detector: fmt.Sprintf("Template (%s)", pattern.Severity),
		})
	}
	return results
}
