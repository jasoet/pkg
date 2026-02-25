// Package base32 provides Crockford Base32 encoding and CRC-10 checksums
// for human-readable, error-correcting identifiers.
//
// The Base32 implementation uses Crockford's alphabet (excludes I, L, O, U)
// and provides case-insensitive decoding with automatic error correction.
//
// The CRC-10 checksum provides 99.9%+ error detection for single character
// errors, transpositions, and double errors.
//
// Example:
//
//	// Encode a value
//	id, err := base32.EncodeBase32(12345, 8)  // "000000C1S", nil
//
//	// Add checksum
//	idWithChecksum, err := base32.AppendChecksum(id)
//
//	// Validate
//	if base32.ValidateChecksum(idWithChecksum) {
//	    // Valid ID
//	}
package base32

import (
	"fmt"
	"math"
	"strings"
)

// Crockford's Base32 alphabet (32 characters, excluding I, L, O, U)
const base32Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// base32DecodeMap maps characters to their Base32 values
// Includes both uppercase and lowercase, and common mistake corrections
var base32DecodeMap = map[rune]int{
	// Digits
	'0': 0, '1': 1, '2': 2, '3': 3, '4': 4,
	'5': 5, '6': 6, '7': 7, '8': 8, '9': 9,

	// Uppercase letters (valid Base32)
	'A': 10, 'B': 11, 'C': 12, 'D': 13, 'E': 14,
	'F': 15, 'G': 16, 'H': 17, 'J': 18, 'K': 19,
	'M': 20, 'N': 21, 'P': 22, 'Q': 23, 'R': 24,
	'S': 25, 'T': 26, 'V': 27, 'W': 28, 'X': 29,
	'Y': 30, 'Z': 31,

	// Lowercase letters (same values as uppercase)
	'a': 10, 'b': 11, 'c': 12, 'd': 13, 'e': 14,
	'f': 15, 'g': 16, 'h': 17, 'j': 18, 'k': 19,
	'm': 20, 'n': 21, 'p': 22, 'q': 23, 'r': 24,
	's': 25, 't': 26, 'v': 27, 'w': 28, 'x': 29,
	'y': 30, 'z': 31,

	// Common mistake corrections
	'I': 1, 'i': 1, // I → 1
	'L': 1, 'l': 1, // L → 1
	'O': 0, 'o': 0, // O → 0
}

// EncodeBase32 encodes an unsigned integer to a Base32 string of specified length.
//
// The encoded value is left-padded with '0's to reach the specified length.
// Uses Crockford's Base32 alphabet (0-9, A-Z excluding I, L, O, U).
//
// Returns an error if the value is too large to fit in the specified length.
//
// Example:
//
//	base32.EncodeBase32(42, 4)     // "001A", nil
//	base32.EncodeBase32(999, 3)    // "0Z7", nil
//	base32.EncodeBase32(32, 2)     // "10", nil
//	base32.EncodeBase32(1024, 2)   // "", error (overflow)
//
// Parameters:
//   - value: The unsigned integer to encode
//   - length: The desired length of the output string
//
// Returns:
//   - A Base32-encoded string of exactly 'length' characters, or "" on error
//   - An error if length <= 0 or the value overflows the specified length
func EncodeBase32(value uint64, length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("length must be positive, got %d", length)
	}

	result := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		result[i] = base32Alphabet[value%32]
		value /= 32
	}

	if value > 0 {
		return "", fmt.Errorf("value too large for %d Base32 characters", length)
	}

	return string(result), nil
}

// DecodeBase32 decodes a Base32 string to an unsigned integer.
//
// Returns an error if the string contains invalid characters.
// Supports case-insensitive input and common error corrections (I→1, L→1, O→0).
//
// Example:
//
//	val, err := base32.DecodeBase32("C1P9")  // 12345, nil
//	val, err := base32.DecodeBase32("c1p9")  // 12345, nil (case-insensitive)
//	val, err := base32.DecodeBase32("C1PO")  // 12345, nil (O→0 correction)
//
// Parameters:
//   - encoded: The Base32-encoded string to decode
//
// Returns:
//   - The decoded unsigned integer value
//   - An error if the input contains invalid characters
func DecodeBase32(encoded string) (uint64, error) {
	if encoded == "" {
		return 0, fmt.Errorf("empty Base32 string")
	}

	var result uint64
	for i, char := range encoded {
		value, ok := base32DecodeMap[char]
		if !ok {
			return 0, fmt.Errorf("invalid Base32 character '%c' at position %d", char, i)
		}
		if result > math.MaxUint64/32 {
			return 0, fmt.Errorf("value overflow at position %d", i)
		}
		next := result*32 + uint64(value)
		if next < result {
			return 0, fmt.Errorf("value overflow at position %d", i)
		}
		result = next
	}

	return result, nil
}

// base32CharToValue converts a single Base32 character to its numeric value.
// Returns -1 if the character is not valid.
func base32CharToValue(char rune) int {
	value, ok := base32DecodeMap[char]
	if !ok {
		return -1
	}
	return value
}

// base32ValueToChar converts a numeric value (0-31) to a Base32 character.
// Returns '0' for invalid values.
func base32ValueToChar(value int) rune {
	if value < 0 || value >= 32 {
		return '0' // Default to '0' for invalid values
	}
	return rune(base32Alphabet[value])
}

// IsValidBase32Char returns true if the character is valid in Base32 encoding.
//
// Valid characters include: 0-9, A-Z (excluding I, L, O, U), and their lowercase
// equivalents. Also accepts I, L, O as they are auto-corrected to 1, 1, 0.
//
// Example:
//
//	base32.IsValidBase32Char('A')  // true
//	base32.IsValidBase32Char('a')  // true
//	base32.IsValidBase32Char('O')  // true (auto-corrected to 0)
//	base32.IsValidBase32Char('U')  // false
func IsValidBase32Char(c rune) bool {
	_, ok := base32DecodeMap[c]
	return ok
}

// NormalizeBase32 normalizes a Base32 string by:
//   - Converting to uppercase
//   - Removing dashes and spaces
//   - Correcting common mistakes (I→1, L→1, O→0)
//
// This function is useful for processing user input to ensure consistency.
//
// Example:
//
//	base32.NormalizeBase32("abc-def")   // "ABCDEF"
//	base32.NormalizeBase32("1O 2I")     // "1021"
//	base32.NormalizeBase32("hell0")     // "HELL0"
// normalizeReplacer performs single-pass replacement of confusable characters and separators.
var normalizeReplacer = strings.NewReplacer(
	"-", "",
	" ", "",
	"I", "1",
	"L", "1",
	"O", "0",
)

func NormalizeBase32(input string) string {
	return normalizeReplacer.Replace(strings.ToUpper(input))
}

// EncodeBase32Compact encodes a value to the minimum number of Base32 characters needed.
//
// Unlike EncodeBase32, this function does not pad the output to a fixed length.
//
// Example:
//
//	base32.EncodeBase32Compact(0)      // "0"
//	base32.EncodeBase32Compact(31)     // "Z"
//	base32.EncodeBase32Compact(32)     // "10"
//	base32.EncodeBase32Compact(12345)  // "C1P9"
func EncodeBase32Compact(value uint64) string {
	if value == 0 {
		return "0"
	}

	var result []byte
	for value > 0 {
		result = append([]byte{base32Alphabet[value%32]}, result...)
		value /= 32
	}

	return string(result)
}
