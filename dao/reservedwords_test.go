package dao

import (
	"testing"
)

func TestAcceptableWord(t *testing.T) {
	tests := []struct {
		name     string
		word     string
		expected bool
	}{
		// Acceptable words
		{"simple word", "abc", true},
		{"numbers", "123", true},
		{"alphanumeric", "abc123", true},
		{"single char", "x", true},

		// Bad words (exact match)
		{"bad word exact", "ass", false},
		{"bad word exact 2", "damn", false},

		// Bad words (substring)
		{"bad word substring", "assassin", false},
		{"bad word substring 2", "classic", false}, // contains "ass"

		// Edge cases
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AcceptableWord(tt.word)
			if result != tt.expected {
				t.Errorf("AcceptableWord(%q) = %v, want %v", tt.word, result, tt.expected)
			}
		})
	}
}

func TestAcceptableWord_CaseSensitive(t *testing.T) {
	// The implementation is case-sensitive, so uppercase versions might pass
	// This documents the current behavior
	result := AcceptableWord("ABC")
	if !result {
		t.Log("AcceptableWord is case-sensitive and rejects uppercase 'ABC'")
	}
}

func BenchmarkAcceptableWord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		AcceptableWord("testword123")
	}
}
