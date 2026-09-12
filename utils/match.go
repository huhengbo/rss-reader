package utils

import "strings"

// MatchStr checks str against keyword patterns and invokes callback for each match.
// A pattern may contain positive terms and terms prefixed with '-' to exclude matches.
func MatchStr(str string, patterns []string, callback func(string)) {
	strFinal := strings.ToLower(strings.TrimSpace(str))

	for _, pattern := range patterns {
		pattern = strings.ToLower(strings.TrimSpace(pattern))
		parts := strings.Split(pattern, " ")
		hasPositive := false
		hasExcluded := false

		for _, part := range parts {
			if strings.HasPrefix(part, "-") {
				hasExcluded = hasExcluded || strings.Contains(strFinal, part[1:])
			} else {
				hasPositive = hasPositive || strings.Contains(strFinal, part)
			}
		}

		if hasPositive && !hasExcluded {
			callback(str)
		}
	}
}
