package scan

import (
	"regexp/syntax"
	"strings"
)

// ExtractKeyword analyzes a regex pattern and returns the longest literal string
// that MUST be present for the regex to match. Returns empty string if none found.
func ExtractKeyword(regexStr string) string {
	// Parse the regex to understand its structure
	re, err := syntax.Parse(regexStr, syntax.Perl)
	if err != nil {
		return ""
	}
	return findBestLiteral(re)
}

func findBestLiteral(re *syntax.Regexp) string {
	switch re.Op {
	case syntax.OpLiteral:
		return string(re.Rune)
	case syntax.OpConcat:
		// Find longest literal in the chain
		var best string
		for _, sub := range re.Sub {
			candidate := findBestLiteral(sub)
			if len(candidate) > len(best) {
				best = candidate
			}
		}

		return best

	case syntax.OpCapture:
		return findBestLiteral(re.Sub[0])

	case syntax.OpPlus: // A+ -> A is required
		return findBestLiteral(re.Sub[0])

	case syntax.OpRepeat: // A{3,5} -> A is required (if min > 0)
		if re.Min > 0 {
			return findBestLiteral(re.Sub[0])
		}
		return ""

	default:
		return ""
	}
}

// IsValidKeyword checks if the keyword is good enough to be an index
func IsValidKeyword(kw string) bool {
	// Must be at least 4 chars to be worth indexing
	if len(kw) < 4 {
		return false
	}

	// Must not be a common garbage string
	kwLower := strings.ToLower(kw)
	common := map[string]bool{
		"http": true, "https": true, "application": true, "password": true,
		"username": true, "token": true, "key": true, "auth": true,
		"bearer": true, "private": true, "public": true, "secret": true,
		"access": true, "stripe": false, "slack": false, "github": false, // Brands are GOOD keywords
	}

	if common[kwLower] {
		return false
	}

	// Filter out keywords composed of repetitive characters
	// e.g. "AAAAA" or "....."
	if isRepetitive(kw) {
		return false
	}

	return true
}

func isRepetitive(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}
