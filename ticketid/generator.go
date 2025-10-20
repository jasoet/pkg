package ticketid

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/jasoet/pkg/v2/base32"
)

// Generate creates a unique ticket ID from the given components
// Format: [EEEEE][DDDDD][CCC][SSSSS][RRRR][XX]
// Example: 01B2M4K6G8N3V9F2HA7XJ5QR
func Generate(
	eventID string,
	eventDate time.Time,
	categoryID string,
	seatID string,
	sequence int,
) (string, error) {
	// Validate inputs
	if err := validateGenerateInputs(eventID, categoryID, seatID, sequence); err != nil {
		return "", err
	}

	// Encode each component to Base32
	eventIDEncoded, err := encodeEventID(eventID)
	if err != nil {
		return "", err
	}

	dateEncoded := encodeEventDate(eventDate)

	categoryEncoded, err := encodeCategoryID(categoryID)
	if err != nil {
		return "", err
	}

	seatIDEncoded, err := encodeSeatID(seatID)
	if err != nil {
		return "", err
	}

	sequenceEncoded := encodeSequence(sequence)

	// Combine all components (without checksum)
	dataWithoutChecksum := eventIDEncoded + dateEncoded + categoryEncoded + seatIDEncoded + sequenceEncoded

	// Add 2-char checksum
	ticketID := base32.AppendChecksum(dataWithoutChecksum)

	return ticketID, nil
}

// GenerateSequence generates a unique sequence number based on timestamp and randomness
// This provides collision resistance while being reproducible if needed
func GenerateSequence() int {
	// Use microseconds from current time
	now := time.Now()
	timePart := now.UnixMicro() % 500000 // 0-499999

	// Add random component (0-499999)
	randomPart, err := rand.Int(rand.Reader, big.NewInt(500000))
	if err != nil {
		// Fallback to nanoseconds if random fails
		randomPart = big.NewInt(int64(now.Nanosecond() % 500000))
	}

	// Combine to create sequence (0-999999)
	sequence := int(timePart + randomPart.Int64())

	// Ensure it fits in our sequence length (max ~1M for 4 Base32 chars)
	if sequence > 1048575 { // 32^4 = 1,048,576
		sequence = sequence % 1048575
	}

	return sequence
}

// Format formats a ticket ID with dashes for readability
// Example: 01B2M4K6G8N3V9F2HA7XJ5QR → 01B2M-4K6G8-N3V-9F2HA-7XJ5-QR
func Format(ticketID string) string {
	if len(ticketID) != TotalLength {
		return ticketID // Return as-is if wrong length
	}

	return fmt.Sprintf("%s-%s-%s-%s-%s-%s",
		ticketID[0:5],   // Event ID
		ticketID[5:10],  // Event Date
		ticketID[10:13], // Category
		ticketID[13:18], // Seat ID
		ticketID[18:22], // Sequence
		ticketID[22:24], // Checksum
	)
}

// encodeEventID encodes an event ID string to 5 Base32 characters
func encodeEventID(eventID string) (string, error) {
	// Try to parse as number
	if num, err := strconv.ParseUint(eventID, 10, 64); err == nil {
		// Max value for 5 Base32 chars: 32^5 = 33,554,432
		if num >= 33554432 {
			return "", NewError(ErrValueOutOfRange, "event ID too large")
		}
		return base32.EncodeBase32(num, EventIDLength), nil
	}

	// If not a number, hash it to a numeric value
	hash := hashStringToUint64(eventID)
	return base32.EncodeBase32(hash%33554432, EventIDLength), nil
}

// encodeEventDate encodes a date to 5 Base32 characters (YYYYMMDD)
func encodeEventDate(date time.Time) string {
	// Format: YYYYMMDD as integer
	dateInt := uint64(date.Year()*10000 + int(date.Month())*100 + date.Day())
	return base32.EncodeBase32(dateInt, EventDateLength)
}

// encodeCategoryID encodes a category ID to 3 Base32 characters
func encodeCategoryID(categoryID string) (string, error) {
	// Try to parse as number
	if num, err := strconv.ParseUint(categoryID, 10, 64); err == nil {
		// Max value for 3 Base32 chars: 32^3 = 32,768
		if num >= 32768 {
			return "", NewError(ErrValueOutOfRange, "category ID too large")
		}
		return base32.EncodeBase32(num, CategoryLength), nil
	}

	// If not a number, hash it
	hash := hashStringToUint64(categoryID)
	return base32.EncodeBase32(hash%32768, CategoryLength), nil
}

// encodeSeatID encodes a seat ID to 5 Base32 characters
func encodeSeatID(seatID string) (string, error) {
	// Try to parse as number
	if num, err := strconv.ParseUint(seatID, 10, 64); err == nil {
		// Max value for 5 Base32 chars: 32^5 = 33,554,432
		if num >= 33554432 {
			return "", NewError(ErrValueOutOfRange, "seat ID too large")
		}
		return base32.EncodeBase32(num, SeatIDLength), nil
	}

	// If not a number, hash it
	hash := hashStringToUint64(seatID)
	return base32.EncodeBase32(hash%33554432, SeatIDLength), nil
}

// encodeSequence encodes a sequence number to 4 Base32 characters
func encodeSequence(sequence int) string {
	// Max value for 4 Base32 chars: 32^4 = 1,048,576
	if sequence < 0 {
		sequence = 0
	}
	if sequence >= 1048576 {
		sequence = sequence % 1048576
	}
	return base32.EncodeBase32(uint64(sequence), SequenceLength)
}

// hashStringToUint64 creates a simple hash of a string for encoding
func hashStringToUint64(s string) uint64 {
	hash := uint64(5381)
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint64(c)
	}
	return hash
}

// validateGenerateInputs validates all inputs for ticket ID generation
func validateGenerateInputs(eventID, categoryID, seatID string, sequence int) error {
	if eventID == "" {
		return NewError(ErrInvalidEventID, "event ID cannot be empty")
	}

	if categoryID == "" {
		return NewError(ErrInvalidCategory, "category ID cannot be empty")
	}

	if seatID == "" {
		return NewError(ErrInvalidSeat, "seat ID cannot be empty")
	}

	if sequence < 0 {
		return NewError(ErrInvalidSequence, "sequence must be non-negative")
	}

	return nil
}
