package base32

import "fmt"

// CRC-10 polynomial for checksum calculation
// x^10 + x^5 + x^4 + x^1 + 1 = 0x233
const crc10Polynomial = 0x233

// CalculateChecksum computes a 2-character Base32 checksum using CRC-10.
//
// The checksum provides 99.9%+ error detection for:
//   - Single character errors
//   - Character transpositions
//   - Double errors
//   - Most insertion/deletion errors
//
// The CRC-10 algorithm processes each Base32 character (5 bits) and produces
// a 10-bit checksum, which is then encoded as 2 Base32 characters.
//
// Returns an error if the input contains invalid Base32 characters.
//
// Example:
//
//	checksum, err := base32.CalculateChecksum("ABC123")  // "XY", nil
//
// Parameters:
//   - data: The Base32 string to checksum (must contain only valid Base32 characters)
//
// Returns:
//   - A 2-character Base32 checksum
//   - An error if the input contains invalid characters
func CalculateChecksum(data string) (string, error) {
	crc := uint16(0)

	// Process each character in the data
	for i, char := range data {
		value := base32CharToValue(char)
		if value < 0 {
			return "", fmt.Errorf("invalid Base32 character '%c' at position %d", char, i)
		}

		// XOR the value into the CRC (shifted left by 5 bits)
		crc ^= uint16(value) << 5

		// Process 5 bits (since Base32 = 5 bits per character)
		for j := 0; j < 5; j++ {
			if crc&0x200 != 0 { // Check if bit 9 is set
				crc = (crc << 1) ^ crc10Polynomial
			} else {
				crc = crc << 1
			}
		}
	}

	// Keep only 10 bits
	crc &= 0x3FF

	// Convert 10 bits to 2 Base32 characters
	// Upper 5 bits → first character
	// Lower 5 bits → second character
	char1 := base32ValueToChar(int((crc >> 5) & 0x1F))
	char2 := base32ValueToChar(int(crc & 0x1F))

	return string([]rune{char1, char2}), nil
}

// ValidateChecksum verifies that the checksum in a string is correct.
//
// Expected format: [data][2 chars checksum]
//
// This function is useful for validating user input or detecting data corruption.
// Returns false if the input is too short or contains invalid Base32 characters.
//
// Example:
//
//	valid := base32.ValidateChecksum("ABC123XY")  // true if XY is correct checksum
//	valid := base32.ValidateChecksum("ABC123ZZ")  // false if ZZ is wrong
//
// Parameters:
//   - input: The string with checksum appended (minimum 3 characters)
//
// Returns:
//   - true if the checksum is valid, false otherwise
func ValidateChecksum(input string) bool {
	if len(input) < 3 {
		return false
	}

	// Split data and checksum
	dataLen := len(input) - 2
	data := input[:dataLen]
	providedChecksum := input[dataLen:]

	// Calculate expected checksum
	expectedChecksum, err := CalculateChecksum(data)
	if err != nil {
		return false
	}

	// Compare checksums (case-insensitive)
	return NormalizeBase32(providedChecksum) == NormalizeBase32(expectedChecksum)
}

// AppendChecksum adds a 2-character checksum to the end of the data.
//
// This is the recommended way to create checksummed strings.
//
// Returns an error if the input contains invalid Base32 characters.
//
// Example:
//
//	id, _ := base32.EncodeBase32(12345, 6)             // "000C1S"
//	idWithChecksum, _ := base32.AppendChecksum(id)     // "000C1SXY"
//
// Parameters:
//   - data: The Base32 string to checksum (must contain only valid Base32 characters)
//
// Returns:
//   - The input string with a 2-character checksum appended
//   - An error if the input contains invalid characters
func AppendChecksum(data string) (string, error) {
	checksum, err := CalculateChecksum(data)
	if err != nil {
		return "", err
	}
	return data + checksum, nil
}

// StripChecksum removes the last 2 characters (checksum) from a string.
//
// Returns an empty string if the input has 2 or fewer characters.
//
// Example:
//
//	data := base32.StripChecksum("ABC123XY")  // "ABC123"
//	data := base32.StripChecksum("AB")        // ""
//
// Parameters:
//   - input: The string with checksum appended
//
// Returns:
//   - The input string without the last 2 characters
func StripChecksum(input string) string {
	if len(input) <= 2 {
		return ""
	}
	return input[:len(input)-2]
}

// ExtractChecksum extracts the last 2 characters (checksum) from a string.
//
// Returns an empty string if the input has fewer than 2 characters.
//
// Example:
//
//	checksum := base32.ExtractChecksum("ABC123XY")  // "XY"
//	checksum := base32.ExtractChecksum("A")         // ""
//
// Parameters:
//   - input: The string with checksum appended
//
// Returns:
//   - The last 2 characters of the input
func ExtractChecksum(input string) string {
	if len(input) < 2 {
		return ""
	}
	return input[len(input)-2:]
}
