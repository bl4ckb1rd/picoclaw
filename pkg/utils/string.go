package utils

import "strings"

// Truncate returns a truncated version of s with at most maxLen runes.
// Handles multi-byte Unicode characters properly.
// If the string is truncated, "..." is appended to indicate truncation.
func Truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	// Reserve 3 chars for "..."
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-3]) + "..."
}

// SplitMessage splits a string into chunks of at most maxLen characters.
// It tries to split at newlines to preserve readability.
func SplitMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			chunks = append(chunks, text)
			break
		}

		// Try to find a good place to split (newline)
		splitIdx := strings.LastIndex(text[:maxLen], "\n")
		if splitIdx == -1 {
			// No newline found, just split at maxLen
			splitIdx = maxLen
		}

		chunks = append(chunks, text[:splitIdx])
		text = strings.TrimSpace(text[splitIdx:])
	}

	return chunks
}
