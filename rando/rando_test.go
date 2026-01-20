package rando

import (
	"strings"
	"testing"
)

func TestRandStrn_Length(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"length 1", 1},
		{"length 5", 5},
		{"length 10", 10},
		{"length 20", 20},
		{"length 0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandStrn(tt.length)
			if len(result) != tt.length {
				t.Errorf("RandStrn(%d) returned string of length %d, want %d", tt.length, len(result), tt.length)
			}
		})
	}
}

func TestRandStrn_ValidCharacters(t *testing.T) {
	validChars := "abcdefghijklmnopqrstuvwxyz0123456789"

	// Generate multiple strings to test
	for i := 0; i < 100; i++ {
		result := RandStrn(20)
		for _, char := range result {
			if !strings.ContainsRune(validChars, char) {
				t.Errorf("RandStrn() returned invalid character: %c", char)
			}
		}
	}
}

func TestRandStrn_Randomness(t *testing.T) {
	// Generate multiple strings and ensure they're not all the same
	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		results[RandStrn(10)] = true
	}

	// With 36 possible characters and length 10, we should get many unique strings
	if len(results) < 90 {
		t.Errorf("RandStrn() generated too few unique strings: %d out of 100", len(results))
	}
}

func TestRandStrn_NoUppercase(t *testing.T) {
	for i := 0; i < 100; i++ {
		result := RandStrn(20)
		if result != strings.ToLower(result) {
			t.Errorf("RandStrn() returned uppercase characters: %s", result)
		}
	}
}

func BenchmarkRandStrn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStrn(10)
	}
}
