package base32

import "fmt"

// CRC-10 polynomial for checksum calculation.
//
// This is CRC-10/ATM: generator x^10 + x^9 + x^5 + x^4 + x + 1, which in the
// normal (non-reflected) representation is 0x233. Note the x^9 term — omitting
// it yields a different polynomial and an incompatible checksum.
const crc10Polynomial = 0x233

// CalculateChecksum computes a 2-character Base32 checksum using CRC-10/ATM.
//
// The checksum provides strong error detection for:
//   - Single character errors
//   - Character transpositions
//   - Double errors
//   - Most insertion/deletion errors
//
// The CRC-10 algorithm processes each Base32 character (5 bits) and produces
// a 10-bit checksum, which is then encoded as 2 Base32 characters.
//
// Leading-zero blind spot: because the CRC register is initialized to zero,
// inserting or deleting leading '0' characters does not change the checksum
// (e.g. CalculateChecksum("C1S") == CalculateChecksum("000C1S")), and any
// all-zero string checksums to "00" and validates. Do not rely on the
// checksum to catch loss or addition of leading zeros; encode identifiers at a
// fixed length (see EncodeBase32) when that matters. This is a compatibility
// contract pinned by the golden vectors and will not change within v3.
//
// Returns ErrEmptyInput for empty input and ErrInvalidCharacter for characters
// outside the Crockford alphabet. Match with errors.Is.
//
// Example:
//
//	checksum, err := base32.CalculateChecksum("ABC123")  // "TF", nil
//
// Parameters:
//   - data: The Base32 string to checksum (must contain only valid Base32 characters)
//
// Returns:
//   - A 2-character Base32 checksum
//   - An error if the input is empty or contains invalid characters
func CalculateChecksum(data string) (string, error) {
	if data == "" {
		return "", fmt.Errorf("empty Base32 input: %w", ErrEmptyInput)
	}

	crc := uint16(0)

	// Process each character in the data
	for i, char := range data {
		value := base32CharToValue(char)
		if value < 0 {
			return "", fmt.Errorf("invalid Base32 character '%c' at position %d: %w", char, i, ErrInvalidCharacter)
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
// The input is normalized via NormalizeBase32 before validation, so
// lowercase, dashed, or spaced input (e.g. "0000-c1p9-q0") validates against
// its normalized form. Clean input is unaffected by normalization.
//
// This function is useful for validating user input or detecting data corruption.
// Returns false if the input is too short or contains invalid Base32 characters.
//
// Example:
//
//	valid := base32.ValidateChecksum("ABC123TF")    // true (TF is the checksum of "ABC123")
//	valid := base32.ValidateChecksum("abc-123-tf")  // true (normalized before validation)
//	valid := base32.ValidateChecksum("ABC123ZZ")    // false (ZZ is the wrong checksum)
//
// Parameters:
//   - input: The string with checksum appended (minimum 3 characters after normalization)
//
// Returns:
//   - true if the checksum is valid, false otherwise
func ValidateChecksum(input string) bool {
	input = NormalizeBase32(input)
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

	// Compare checksums (input is already normalized above)
	return providedChecksum == expectedChecksum
}

// AppendChecksum adds a 2-character checksum to the end of the data.
//
// This is the recommended way to create checksummed strings.
//
// The input is normalized via NormalizeBase32 before the checksum is
// computed, so lowercase, dashed, or spaced input (e.g. "0000-c1p9") works;
// the returned string is always the normalized data plus its checksum.
// Clean input is unaffected by normalization.
//
// Returns ErrEmptyInput if the input is empty after normalization (e.g. "---",
// which normalizes to "") and ErrInvalidCharacter if it contains characters
// outside the Crockford alphabet. Match with errors.Is.
//
// Example:
//
//	id, _ := base32.EncodeBase32(12345, 6)             // "000C1S"
//	idWithChecksum, _ := base32.AppendChecksum(id)     // "000C1S69"
//	withDashes, _ := base32.AppendChecksum("0000-c1p9") // "0000C1P9Q0" (normalized)
//
// Parameters:
//   - data: The Base32 string to checksum (normalized before checksumming)
//
// Returns:
//   - The normalized input string with a 2-character checksum appended
//   - An error if the normalized input is empty or contains invalid characters
func AppendChecksum(data string) (string, error) {
	normalized := NormalizeBase32(data)
	if normalized == "" {
		return "", fmt.Errorf("input %q is empty after normalization: %w", data, ErrEmptyInput)
	}
	checksum, err := CalculateChecksum(normalized)
	if err != nil {
		return "", err
	}
	return normalized + checksum, nil
}

// StripChecksum removes the last 2 characters (checksum) from a string.
//
// The input is normalized via NormalizeBase32 first, mirroring
// AppendChecksum/ValidateChecksum. This is required for correctness: a
// checksummed string that validated in dashed/lowercase form (e.g.
// "0000-c1p9-q0") strips to the normalized payload ("0000C1P9") rather than a
// byte-sliced fragment of the raw input. Normalizing also avoids splitting a
// multibyte rune when slicing.
//
// Returns an empty string if the normalized input has 2 or fewer characters.
//
// Example:
//
//	data := base32.StripChecksum("ABC123TF")     // "ABC123"
//	data := base32.StripChecksum("0000-c1p9-q0") // "0000C1P9" (normalized first)
//	data := base32.StripChecksum("AB")           // ""
//
// Parameters:
//   - input: The string with checksum appended (normalized before stripping)
//
// Returns:
//   - The normalized input string without the last 2 characters
func StripChecksum(input string) string {
	input = NormalizeBase32(input)
	if len(input) <= 2 {
		return ""
	}
	return input[:len(input)-2]
}

// ExtractChecksum extracts the last 2 characters (checksum) from a string.
//
// The input is normalized via NormalizeBase32 first, mirroring
// AppendChecksum/ValidateChecksum, so the checksum of a validated
// dashed/lowercase string (e.g. "0000-c1p9-q0" → "Q0") is returned rather than
// a byte-sliced fragment of the raw input. Normalizing also avoids splitting a
// multibyte rune when slicing.
//
// Returns an empty string if the normalized input has fewer than 2 characters.
//
// Example:
//
//	checksum := base32.ExtractChecksum("ABC123TF")     // "TF"
//	checksum := base32.ExtractChecksum("0000-c1p9-q0") // "Q0" (normalized first)
//	checksum := base32.ExtractChecksum("A")            // ""
//
// Parameters:
//   - input: The string with checksum appended (normalized before extracting)
//
// Returns:
//   - The last 2 characters of the normalized input
func ExtractChecksum(input string) string {
	input = NormalizeBase32(input)
	if len(input) < 2 {
		return ""
	}
	return input[len(input)-2:]
}
