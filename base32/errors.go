package base32

import "errors"

// Sentinel errors returned (wrapped) by this package. Match them with
// errors.Is rather than string-comparing messages; the wrapped errors carry
// human-readable detail (offending character, position, original input).
var (
	// ErrEmptyInput is returned when an operation requires Base32 content but
	// the input is empty, or empty after normalization (separator-only input
	// such as "---").
	ErrEmptyInput = errors.New("empty Base32 input")

	// ErrInvalidCharacter is returned when the input contains a character that
	// is not part of the Crockford Base32 alphabet after normalization.
	ErrInvalidCharacter = errors.New("invalid Base32 character")

	// ErrOverflow is returned when a Base32 string decodes to a value that does
	// not fit in a uint64.
	ErrOverflow = errors.New("Base32 value overflows uint64")

	// ErrValueTooLarge is returned when a value cannot be encoded within the
	// requested fixed length.
	ErrValueTooLarge = errors.New("value too large for requested Base32 length")
)
