package base32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateChecksum(t *testing.T) {
	tests := []struct {
		name string
		data string
		// We don't check exact checksum values here since they depend on the algorithm
		// Instead we verify properties: same input = same output, different input = different output
	}{
		{"simple", "ABC123"},
		{"zeros", "000000"},
		{"all letters", "ABCDEFGH"},
		{"all numbers", "12345678"},
		{"mixed", "A1B2C3D4"},
		{"single char", "A"},
		{"two chars", "AB"},
		{"long string", "0123456789ABCDEFGHIJKLMNPQRSTUV"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum := CalculateChecksum(tt.data)

			// Checksum should always be 2 characters
			assert.Len(t, checksum, 2)

			// Checksum should be deterministic
			checksum2 := CalculateChecksum(tt.data)
			assert.Equal(t, checksum, checksum2)

			// Checksum should only contain valid Base32 characters
			for _, char := range checksum {
				assert.True(t, IsValidBase32Char(char))
			}
		})
	}
}

func TestCalculateChecksumDifferentInputs(t *testing.T) {
	// Different inputs should produce different checksums (with high probability)
	data1 := "ABC123"
	data2 := "ABC124"
	data3 := "XYZ789"

	checksum1 := CalculateChecksum(data1)
	checksum2 := CalculateChecksum(data2)
	checksum3 := CalculateChecksum(data3)

	assert.NotEqual(t, checksum1, checksum2, "different data should have different checksums")
	assert.NotEqual(t, checksum1, checksum3, "different data should have different checksums")
	assert.NotEqual(t, checksum2, checksum3, "different data should have different checksums")
}

func TestValidateChecksum(t *testing.T) {
	// Generate valid checksummed strings
	testData := []string{"ABC123", "000000", "HELLO", "12345", "ZYXWVU"}

	for _, data := range testData {
		t.Run(data, func(t *testing.T) {
			checksum := CalculateChecksum(data)
			fullString := data + checksum

			// Valid checksum should pass
			assert.True(t, ValidateChecksum(fullString))

			// Modified checksum should fail
			if len(fullString) > 2 {
				modified := fullString[:len(fullString)-1] + "X"
				assert.False(t, ValidateChecksum(modified))
			}
		})
	}
}

func TestValidateChecksumEdgeCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"empty string", "", false},
		{"single char", "A", false},
		{"two chars", "AB", false},
		{"minimum valid", "ABC", true}, // "A" + 2-char checksum
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For the minimum valid case, we need to actually compute it
			if tt.name == "minimum valid" {
				checksum := CalculateChecksum("A")
				input := "A" + checksum
				assert.True(t, ValidateChecksum(input))
			} else {
				assert.Equal(t, tt.valid, ValidateChecksum(tt.input))
			}
		})
	}
}

func TestValidateChecksumCaseInsensitive(t *testing.T) {
	data := "ABC123"
	checksum := CalculateChecksum(data)
	fullString := data + checksum

	// Test case insensitivity
	tests := []string{
		fullString,
		toLower(fullString),
		mixCase(fullString),
	}

	for _, test := range tests {
		assert.True(t, ValidateChecksum(test), "checksum validation should be case-insensitive")
	}
}

func TestAppendChecksum(t *testing.T) {
	tests := []string{"ABC123", "000000", "HELLO", "12345"}

	for _, data := range tests {
		t.Run(data, func(t *testing.T) {
			result := AppendChecksum(data)

			// Result should be data + 2 chars
			assert.Len(t, result, len(data)+2)

			// Should start with original data
			assert.Equal(t, data, result[:len(data)])

			// Should validate correctly
			assert.True(t, ValidateChecksum(result))
		})
	}
}

func TestStripChecksum(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal", "ABC123XY", "ABC123"},
		{"short", "ABC", "A"},
		{"two chars", "AB", ""},
		{"one char", "A", ""},
		{"empty", "", ""},
		{"long", "0123456789ABCDEFGH", "0123456789ABCDEF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripChecksum(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractChecksum(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"normal", "ABC123XY", "XY"},
		{"short", "ABC", "BC"},
		{"two chars", "AB", "AB"},
		{"one char", "A", ""},
		{"empty", "", ""},
		{"long", "0123456789ABCDEFGH", "GH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractChecksum(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendStripRoundTrip(t *testing.T) {
	testData := []string{"ABC123", "000000", "HELLO", "12345", "ZYXWVU"}

	for _, data := range testData {
		t.Run(data, func(t *testing.T) {
			withChecksum := AppendChecksum(data)
			stripped := StripChecksum(withChecksum)

			assert.Equal(t, data, stripped)
		})
	}
}

func TestChecksumErrorDetection(t *testing.T) {
	originalData := "ABC123"
	checksummed := AppendChecksum(originalData)

	// Test single character error detection
	t.Run("single char error", func(t *testing.T) {
		// Corrupt one character in the data
		corrupted := "XBC123" + checksummed[len(originalData):]
		assert.False(t, ValidateChecksum(corrupted))
	})

	// Test transposition detection
	t.Run("transposition", func(t *testing.T) {
		// Swap two adjacent characters
		if len(originalData) >= 2 {
			chars := []rune(checksummed)
			chars[0], chars[1] = chars[1], chars[0]
			transposed := string(chars)
			assert.False(t, ValidateChecksum(transposed))
		}
	})

	// Test double error detection
	t.Run("double error", func(t *testing.T) {
		// Corrupt two characters
		corrupted := "XYC123" + checksummed[len(originalData):]
		assert.False(t, ValidateChecksum(corrupted))
	})
}

func TestInvalidCharacterHandling(t *testing.T) {
	// CalculateChecksum should handle invalid characters gracefully (treat as 0)
	checksum1 := CalculateChecksum("ABC")
	checksum2 := CalculateChecksum("A#C") // # is invalid, treated as 0
	checksum3 := CalculateChecksum("A0C") // 0 is valid

	// #treated as 0, so checksum2 should equal checksum3
	assert.Equal(t, checksum3, checksum2)
	// But should differ from ABC
	assert.NotEqual(t, checksum1, checksum2)
}

// Helper functions
func toLower(s string) string {
	result := ""
	for _, char := range s {
		if char >= 'A' && char <= 'Z' {
			result += string(char + 32)
		} else {
			result += string(char)
		}
	}
	return result
}

func mixCase(s string) string {
	result := ""
	for i, char := range s {
		if i%2 == 0 && char >= 'A' && char <= 'Z' {
			result += string(char + 32) // to lowercase
		} else if i%2 == 1 && char >= 'a' && char <= 'z' {
			result += string(char - 32) // to uppercase
		} else {
			result += string(char)
		}
	}
	return result
}

// Benchmark tests
func BenchmarkCalculateChecksum(b *testing.B) {
	data := "0123456789ABCDEF"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CalculateChecksum(data)
	}
}

func BenchmarkValidateChecksum(b *testing.B) {
	data := "0123456789ABCDEF"
	checksummed := AppendChecksum(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ValidateChecksum(checksummed)
	}
}

func BenchmarkAppendChecksum(b *testing.B) {
	data := "0123456789ABCDEF"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		AppendChecksum(data)
	}
}
