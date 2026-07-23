package base32

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Golden regression tests.
//
// Every expected value below was captured by RUNNING the actual functions
// (never copied from documentation). If a change to the encoding, alphabet,
// or CRC-10 implementation alters any of these outputs, these tests fail —
// that is intentional: encoded values and checksums are a compatibility
// contract with anything stored or printed by earlier versions.

func TestGoldenEncodeBase32(t *testing.T) {
	vectors := []struct {
		value  uint64
		length int
		want   string
	}{
		{0, 6, "000000"},
		{42, 4, "001A"},
		{42, 2, "1A"},
		{31, 2, "0Z"},
		{32, 2, "10"},
		{999, 3, "0Z7"},
		{12345, 8, "00000C1S"},
		{12345, 6, "000C1S"},
		{12345, 4, "0C1S"},
		{789, 4, "00RN"},
		{9999, 4, "09RF"},
		{20251231, 6, "0KA0JZ"},
		{^uint64(0), 13, "FZZZZZZZZZZZZ"},
	}

	for _, v := range vectors {
		got, err := EncodeBase32(v.value, v.length)
		require.NoError(t, err)
		assert.Equal(t, v.want, got, "EncodeBase32(%d, %d)", v.value, v.length)
	}
}

func TestGoldenEncodeBase32Compact(t *testing.T) {
	vectors := []struct {
		value uint64
		want  string
	}{
		{0, "0"},
		{31, "Z"},
		{32, "10"},
		{42, "1A"},
		{789, "RN"},
		{9999, "9RF"},
		{12345, "C1S"},
		{123456, "3RJ0"},
		{123456789, "3NQK8N"},
	}

	for _, v := range vectors {
		assert.Equal(t, v.want, EncodeBase32Compact(v.value), "EncodeBase32Compact(%d)", v.value)
	}
}

func TestGoldenDecodeBase32(t *testing.T) {
	vectors := []struct {
		encoded string
		want    uint64
	}{
		{"C1S", 12345},
		{"c1s", 12345}, // case-insensitive
		{"I0", 32},     // I→1 correction
		{"3NQK8N", 123456789},
		{"00000C1S", 12345},
	}

	for _, v := range vectors {
		got, err := DecodeBase32(v.encoded)
		require.NoError(t, err)
		assert.Equal(t, v.want, got, "DecodeBase32(%q)", v.encoded)
	}
}

func TestGoldenNormalizeBase32(t *testing.T) {
	vectors := []struct {
		input string
		want  string
	}{
		{"abc-def", "ABCDEF"},
		{"1O 2I", "1021"},
		{"hell0", "HE110"},
		{"ABC DEF", "ABCDEF"},
		{"ABCD-EFGH-IJ", "ABCDEFGH1J"},
		{"ABCDEF", "ABCDEF"}, // clean input is unaffected
		{"0000-C1P9", "0000C1P9"},
	}

	for _, v := range vectors {
		assert.Equal(t, v.want, NormalizeBase32(v.input), "NormalizeBase32(%q)", v.input)
	}
}

func TestGoldenChecksums(t *testing.T) {
	vectors := []struct {
		data         string
		checksum     string
		withChecksum string
	}{
		{"ABC123", "TF", "ABC123TF"},
		{"C1S", "69", "C1S69"},
		{"000C1S", "69", "000C1S69"},
		{"0000C1P9", "Q0", "0000C1P9Q0"},
		{"TEST123", "WB", "TEST123WB"},
		{"001A", "9P", "001A9P"},
		{"000000", "00", "00000000"},
		{"1A00RN", "7D", "1A00RN7D"},
		{"09RF", "ZB", "09RFZB"},
		{"3RJ0", "N1", "3RJ0N1"},
		{"3NQK8N", "3M", "3NQK8N3M"},
	}

	for _, v := range vectors {
		checksum, err := CalculateChecksum(v.data)
		require.NoError(t, err)
		assert.Equal(t, v.checksum, checksum, "CalculateChecksum(%q)", v.data)

		withChecksum, err := AppendChecksum(v.data)
		require.NoError(t, err)
		assert.Equal(t, v.withChecksum, withChecksum, "AppendChecksum(%q)", v.data)

		assert.True(t, ValidateChecksum(v.withChecksum), "ValidateChecksum(%q)", v.withChecksum)

		assert.Equal(t, v.checksum, ExtractChecksum(v.withChecksum), "ExtractChecksum(%q)", v.withChecksum)
		assert.Equal(t, v.data, StripChecksum(v.withChecksum), "StripChecksum(%q)", v.withChecksum)
	}

	assert.False(t, ValidateChecksum("ABC123ZZ"), "wrong checksum must not validate")
}

func TestGoldenRoundTrip(t *testing.T) {
	// Encode → append checksum → validate → strip → decode must round-trip.
	for _, value := range []uint64{0, 1, 42, 12345, 123456789} {
		encoded, err := EncodeBase32(value, 8)
		require.NoError(t, err)

		withChecksum, err := AppendChecksum(encoded)
		require.NoError(t, err)

		require.True(t, ValidateChecksum(withChecksum), "ValidateChecksum(%q)", withChecksum)

		decoded, err := DecodeBase32(StripChecksum(withChecksum))
		require.NoError(t, err)
		assert.Equal(t, value, decoded)
	}
}

// TestAppendChecksum_Normalizes pins the normalization contract:
// AppendChecksum normalizes its input via NormalizeBase32 before computing
// the checksum, so lowercase and dashed/spaced input works and the returned
// string is always in normalized (uppercase, separator-free) form.
// Clean input is unaffected.
func TestAppendChecksum_Normalizes(t *testing.T) {
	// Lowercase input: succeeds and returns normalized output.
	got, err := AppendChecksum("0000c1p9")
	require.NoError(t, err)
	assert.Equal(t, "0000C1P9Q0", got)

	// Dashed input: dashes are stripped, checksum computed over normalized data.
	got, err = AppendChecksum("0000-C1P9")
	require.NoError(t, err)
	assert.Equal(t, "0000C1P9Q0", got)

	// Mixed lowercase + dashes + spaces.
	got, err = AppendChecksum("abc-123")
	require.NoError(t, err)
	assert.Equal(t, "ABC123TF", got)

	// Prefixed, dashed identifiers (e.g. order IDs) work out of the box.
	got, err = AppendChecksum("ORD-6HG4K2N0-00C1S")
	require.NoError(t, err)
	expected, err := AppendChecksum("ORD6HG4K2N000C1S")
	require.NoError(t, err)
	assert.Equal(t, expected, got)

	// Clean input is unaffected by normalization.
	got, err = AppendChecksum("ABC123")
	require.NoError(t, err)
	assert.Equal(t, "ABC123TF", got)
}

// TestValidateChecksum_Normalizes pins the normalization contract:
// ValidateChecksum normalizes its input via NormalizeBase32 before
// validating, so lowercase and dashed/spaced checksummed strings validate.
// Clean input is unaffected.
func TestValidateChecksum_Normalizes(t *testing.T) {
	// Dashed + lowercase form of a valid checksummed string.
	assert.True(t, ValidateChecksum("0000-c1p9-q0"))
	assert.True(t, ValidateChecksum("abc123tf"))
	assert.True(t, ValidateChecksum("ABC123-TF"))

	// Clean input still validates.
	assert.True(t, ValidateChecksum("ABC123TF"))

	// Normalization does not make wrong checksums valid.
	assert.False(t, ValidateChecksum("ABC123-ZZ"))
}
