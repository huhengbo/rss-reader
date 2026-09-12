package feed

import "strings"

// MatchTitle checks a title against keyword patterns and invokes callback for each match.
// A pattern may contain positive terms and terms prefixed with '-' to exclude matches.
func MatchTitle(title string, patterns []string, callback func(string)) {
	finalTitle := strings.ToLower(strings.TrimSpace(title))

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		parts := strings.Split(pattern, " ")
		hasPositive := false
		hasExcluded := false

		for _, part := range parts {
			if strings.HasPrefix(part, "-") {
				hasExcluded = hasExcluded || strings.Contains(finalTitle, part[1:])
			} else {
				hasPositive = hasPositive || strings.Contains(finalTitle, part)
			}
		}

		if hasPositive && !hasExcluded {
			callback(title)
		}
	}
}
