package base32

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSentinelErrors pins the errors.Is-matchable sentinel contract so callers
// can branch on error kind instead of string-matching messages.
func TestSentinelErrors(t *testing.T) {
	t.Run("DecodeBase32 empty is ErrEmptyInput", func(t *testing.T) {
		_, err := DecodeBase32("")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyInput)
	})

	t.Run("DecodeBase32 separator-only is ErrEmptyInput", func(t *testing.T) {
		_, err := DecodeBase32("---")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyInput)
	})

	t.Run("DecodeBase32 invalid char is ErrInvalidCharacter", func(t *testing.T) {
		_, err := DecodeBase32("A#C")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCharacter)
	})

	t.Run("DecodeBase32 overflow is ErrOverflow", func(t *testing.T) {
		_, err := DecodeBase32("ZZZZZZZZZZZZZZ") // 14 Z's overflow uint64
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrOverflow)
	})

	t.Run("EncodeBase32 overflow is ErrValueTooLarge", func(t *testing.T) {
		_, err := EncodeBase32(1024, 2)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrValueTooLarge)
	})

	t.Run("CalculateChecksum empty is ErrEmptyInput", func(t *testing.T) {
		_, err := CalculateChecksum("")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyInput)
	})

	t.Run("CalculateChecksum invalid char is ErrInvalidCharacter", func(t *testing.T) {
		_, err := CalculateChecksum("A#C")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidCharacter)
	})

	t.Run("AppendChecksum empty-after-normalization is ErrEmptyInput", func(t *testing.T) {
		_, err := AppendChecksum("---")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyInput)
		// Message should not misleadingly call the non-empty input "empty".
		assert.Contains(t, err.Error(), "normaliz")
	})
}

// TestStripExtractNormalize pins the fix for the normalization mismatch:
// StripChecksum/ExtractChecksum must normalize their input to mirror the
// Append/Validate contract, so a validated (dashed/lowercase) string yields
// the correct payload and checksum rather than a byte-sliced corruption.
func TestStripExtractNormalize(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantStrip string
		wantCheck string
	}{
		{"dashed lowercase", "0000-c1p9-q0", "0000C1P9", "Q0"},
		{"trailing dash", "C1S69-", "C1S", "69"},
		{"lowercase", "abc123tf", "ABC123", "TF"},
		{"spaced", "ABC123 TF", "ABC123", "TF"},
		{"clean unaffected", "ABC123TF", "ABC123", "TF"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantStrip, StripChecksum(tt.input))
			assert.Equal(t, tt.wantCheck, ExtractChecksum(tt.input))
		})
	}
}

// TestValidateStripDecodeRoundTrip proves the Validate → Strip → Decode
// pipeline works on dashed/lowercase input end-to-end without corruption.
func TestValidateStripDecodeRoundTrip(t *testing.T) {
	// "0000-c1p9-q0" is the dashed/lowercase form of "0000C1P9Q0"
	// (data "0000C1P9" + checksum "Q0").
	input := "0000-c1p9-q0"

	require.True(t, ValidateChecksum(input), "dashed/lowercase checksummed string must validate")

	stripped := StripChecksum(input)
	require.Equal(t, "0000C1P9", stripped)

	decoded, err := DecodeBase32(stripped)
	require.NoError(t, err)

	// Decoding the clean form must yield the same value.
	want, err := DecodeBase32("0000C1P9")
	require.NoError(t, err)
	assert.Equal(t, want, decoded)
}

// TestDecodeBase32StripsSeparators pins that DecodeBase32 ignores hyphens and
// whitespace (Crockford spec) so the checksum entry points and Decode agree.
func TestDecodeBase32StripsSeparators(t *testing.T) {
	clean, err := DecodeBase32("000C1S")
	require.NoError(t, err)

	dashed, err := DecodeBase32("000-c1s")
	require.NoError(t, err)
	assert.Equal(t, clean, dashed)

	spaced, err := DecodeBase32("00 0C 1S")
	require.NoError(t, err)
	assert.Equal(t, clean, spaced)
}

// Fuzz targets: parsing untrusted input must never panic, and encode/decode
// must round-trip.

func FuzzDecodeBase32(f *testing.F) {
	seeds := []string{"", "C1S", "c1s", "0000-c1p9-q0", "ZZZZZZZZZZZZZZ", "A#C", "---", "\x00", "U"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		val, err := DecodeBase32(s)
		if err != nil {
			return
		}
		// On success, the decoded value must re-encode and decode back equal.
		reencoded := EncodeBase32Compact(val)
		back, err := DecodeBase32(reencoded)
		require.NoError(t, err)
		assert.Equal(t, val, back)
	})
}

func FuzzValidateChecksum(f *testing.F) {
	seeds := []string{"", "AB", "ABC123TF", "abc-123-tf", "ABC123ZZ", "---0-0-0-", "\xff\xfe"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		// Must never panic; the result is only required to be a bool.
		_ = ValidateChecksum(s)
	})
}

func FuzzEncodeDecode(f *testing.F) {
	seeds := []uint64{0, 1, 31, 32, 42, 12345, 123456789, ^uint64(0)}
	for _, v := range seeds {
		f.Add(v)
	}
	f.Fuzz(func(t *testing.T, v uint64) {
		// Compact round-trip.
		decoded, err := DecodeBase32(EncodeBase32Compact(v))
		require.NoError(t, err)
		assert.Equal(t, v, decoded)

		// Fixed-length round-trip (13 chars fits any uint64).
		encoded, err := EncodeBase32(v, 13)
		require.NoError(t, err)
		decoded, err = DecodeBase32(encoded)
		require.NoError(t, err)
		assert.Equal(t, v, decoded)

		// Checksum round-trip: append then validate then strip then decode.
		withChecksum, err := AppendChecksum(encoded)
		require.NoError(t, err)
		require.True(t, ValidateChecksum(withChecksum))
		decoded, err = DecodeBase32(StripChecksum(withChecksum))
		require.NoError(t, err)
		assert.Equal(t, v, decoded)
	})
}
