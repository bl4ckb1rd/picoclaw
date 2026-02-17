package channels

import (
	"testing"

	"github.com/sipeed/picoclaw/pkg/utils"
)

func TestSplitMessage(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		maxLen   int
		expected []string
	}{
		{
			name:     "Short message",
			text:     "Hello world",
			maxLen:   20,
			expected: []string{"Hello world"},
		},
		{
			name:     "Split at exact length",
			text:     "1234567890",
			maxLen:   5,
			expected: []string{"12345", "67890"},
		},
		{
			name:     "Split at newline",
			text:     "Line 1\nLine 2\nLine 3",
			maxLen:   10,
			expected: []string{"Line 1", "Line 2", "Line 3"},
		},
		{
			name:     "Long word without newline",
			text:     "VeryLongWordThatExceedsLimit",
			maxLen:   10,
			expected: []string{"VeryLongWo", "rdThatExce", "edsLimit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.SplitMessage(tt.text, tt.maxLen)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d chunks, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("chunk %d: expected %q, got %q", i, tt.expected[i], result[i])
				}
			}
		})
	}
}
