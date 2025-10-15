package base32

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncodeBase32(t *testing.T) {
	tests := []struct {
		name   string
		value  uint64
		length int
		want   string
	}{
		{"zero", 0, 6, "000000"},
		{"small value", 42, 4, "001A"},
		{"max 5 bits", 31, 2, "0Z"},
		{"32", 32, 2, "10"},
		{"example", 12345, 6, "000C1S"},
		{"large value", 999, 3, "0Z7"},
		{"length 1", 5, 1, "5"},
		{"exact fit", 1023, 2, "ZZ"},
		{"zero length", 42, 0, ""},
		{"negative length", 42, -1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeBase32(tt.value, tt.length)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDecodeBase32(t *testing.T) {
	tests := []struct {
		name    string
		encoded string
		want    uint64
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"zero padded", "000000", 0, false},
		{"small value", "1A", 42, false},
		{"max 5 bits", "Z", 31, false},
		{"example", "C1S", 12345, false},
		{"large value", "ZZ", 1023, false},
		{"case insensitive upper", "ABC", 10604, false},
		{"case insensitive lower", "abc", 10604, false},
		{"case insensitive mixed", "AbC", 10604, false},
		{"with I correction", "1I", 33, false},  // I→1
		{"with L correction", "1L", 33, false},  // L→1
		{"with O correction", "1O", 32, false},  // O→0
		{"empty string", "", 0, true},
		{"invalid char U", "U", 0, true},
		{"invalid char #", "#", 0, true},
		{"valid then invalid", "A#", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeBase32(tt.encoded)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	// Property-based testing: encode then decode should return original value
	testCases := []uint64{
		0, 1, 31, 32, 42, 100, 999, 1000, 1023, 1024,
		10000, 32767, 32768, 65535, 65536,
		1000000, 9999999,
	}

	for _, original := range testCases {
		t.Run(string(rune(original)), func(t *testing.T) {
			encoded := EncodeBase32(original, 10)
			decoded, err := DecodeBase32(encoded)

			assert.NoError(t, err)
			assert.Equal(t, original, decoded)
		})
	}
}

func TestEncodeBase32Compact(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		want  string
	}{
		{"zero", 0, "0"},
		{"one", 1, "1"},
		{"31", 31, "Z"},
		{"32", 32, "10"},
		{"42", 42, "1A"},
		{"12345", 12345, "C1S"},
		{"large value", 99999, "31MZ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EncodeBase32Compact(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidBase32Char(t *testing.T) {
	validChars := "0123456789ABCDEFGHJKMNPQRSTVWXYZabcdefghjkmnpqrstvwxyzILOilo"
	for _, char := range validChars {
		t.Run(string(char), func(t *testing.T) {
			assert.True(t, IsValidBase32Char(char), "char %c should be valid", char)
		})
	}

	invalidChars := "U u @ # $ % ^ & * ( ) - + = [ ] { } | \\ / ? < > , . ; : ' \" ~"
	for _, char := range invalidChars {
		if char == ' ' {
			continue // space is filtered by normalize, tested separately
		}
		t.Run(string(char), func(t *testing.T) {
			assert.False(t, IsValidBase32Char(char), "char %c should be invalid", char)
		})
	}
}

func TestNormalizeBase32(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lowercase", "abc", "ABC"},
		{"uppercase", "ABC", "ABC"},
		{"mixed case", "AbC", "ABC"},
		{"with dashes", "ABC-DEF", "ABCDEF"},
		{"with spaces", "ABC DEF", "ABCDEF"},
		{"with dashes and spaces", "AB-CD EF", "ABCDEF"},
		{"I to 1", "1I2", "112"},
		{"L to 1", "1L2", "112"},
		{"O to 0", "1O2", "102"},
		{"multiple corrections", "HELLO", "HE110"},
		{"complex", "ab-CD iL o9", "ABCD1109"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeBase32(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBase32CharToValue(t *testing.T) {
	tests := []struct {
		char rune
		want int
	}{
		{'0', 0},
		{'9', 9},
		{'A', 10},
		{'Z', 31},
		{'a', 10},
		{'z', 31},
		{'I', 1},
		{'L', 1},
		{'O', 0},
		{'U', -1},
		{'#', -1},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			got := base32CharToValue(tt.char)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBase32ValueToChar(t *testing.T) {
	tests := []struct {
		value int
		want  rune
	}{
		{0, '0'},
		{9, '9'},
		{10, 'A'},
		{31, 'Z'},
		{-1, '0'},  // invalid, returns default
		{32, '0'},  // invalid, returns default
		{100, '0'}, // invalid, returns default
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.value)), func(t *testing.T) {
			got := base32ValueToChar(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Benchmark tests
func BenchmarkEncodeBase32(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeBase32(12345, 10)
	}
}

func BenchmarkDecodeBase32(b *testing.B) {
	encoded := "000C1S"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DecodeBase32(encoded)
	}
}

func BenchmarkEncodeBase32Compact(b *testing.B) {
	for i := 0; i < b.N; i++ {
		EncodeBase32Compact(12345)
	}
}

func BenchmarkNormalizeBase32(b *testing.B) {
	input := "ABC-DEF-GHI-JKL"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NormalizeBase32(input)
	}
}
