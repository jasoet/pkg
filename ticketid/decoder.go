package ticketid

import (
	"fmt"
	"strings"
	"time"

	"github.com/jasoet/pkg/v2/base32"
)

// DecodeTicketID decodes a ticket ID string into its components
// Validates checksum and extracts all encoded data
func DecodeTicketID(ticketID string) (*TicketID, error) {
	// Normalize input (remove dashes, convert to uppercase)
	ticketID = base32.NormalizeBase32(ticketID)

	// Validate length
	if len(ticketID) != TotalLength {
		return nil, NewErrorWithDetails(
			ErrInvalidLength,
			"ticket ID must be exactly 24 characters",
			map[string]interface{}{
				"expected": TotalLength,
				"actual":   len(ticketID),
			},
		)
	}

	// Validate checksum first
	if !base32.ValidateChecksum(ticketID) {
		return nil, NewError(ErrInvalidChecksum, "ticket ID checksum validation failed")
	}

	// Extract components
	eventIDPart := ticketID[0:5]
	datePart := ticketID[5:10]
	categoryPart := ticketID[10:13]
	seatIDPart := ticketID[13:18]
	sequencePart := ticketID[18:22]
	checksumPart := ticketID[22:24]

	// Decode each component
	eventIDNum, err := base32.DecodeBase32(eventIDPart)
	if err != nil {
		return nil, WrapError(ErrDecodeFailed, "failed to decode event ID", err)
	}

	dateNum, err := base32.DecodeBase32(datePart)
	if err != nil {
		return nil, WrapError(ErrDecodeFailed, "failed to decode event date", err)
	}

	categoryNum, err := base32.DecodeBase32(categoryPart)
	if err != nil {
		return nil, WrapError(ErrDecodeFailed, "failed to decode category ID", err)
	}

	seatIDNum, err := base32.DecodeBase32(seatIDPart)
	if err != nil {
		return nil, WrapError(ErrDecodeFailed, "failed to decode seat ID", err)
	}

	sequenceNum, err := base32.DecodeBase32(sequencePart)
	if err != nil {
		return nil, WrapError(ErrDecodeFailed, "failed to decode sequence", err)
	}

	// Parse date from YYYYMMDD format
	eventDate, err := parseDateFromEncoded(dateNum)
	if err != nil {
		return nil, WrapError(ErrInvalidDate, "invalid event date", err)
	}

	// Create TicketID struct
	ticket := &TicketID{
		EventID:    formatEventID(eventIDNum),
		EventDate:  eventDate,
		CategoryID: formatCategoryID(categoryNum),
		SeatID:     formatSeatID(seatIDNum),
		Sequence:   int(sequenceNum),
		EncodedID:  ticketID,
		Checksum:   checksumPart,
	}

	return ticket, nil
}

// ParseTicketID is an alias for DecodeTicketID
func ParseTicketID(ticketID string) (*TicketID, error) {
	return DecodeTicketID(ticketID)
}

// parseDateFromEncoded parses a date from YYYYMMDD encoded as uint64
func parseDateFromEncoded(dateNum uint64) (time.Time, error) {
	// Extract year, month, day
	year := int(dateNum / 10000)
	month := int((dateNum % 10000) / 100)
	day := int(dateNum % 100)

	// Validate ranges
	if year < 1970 || year > 2100 {
		return time.Time{}, NewError(ErrInvalidDate, "year out of range (1970-2100)")
	}
	if month < 1 || month > 12 {
		return time.Time{}, NewError(ErrInvalidDate, "month out of range (1-12)")
	}
	if day < 1 || day > 31 {
		return time.Time{}, NewError(ErrInvalidDate, "day out of range (1-31)")
	}

	// Create time object
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// Validate the date is real (e.g., not Feb 30)
	if date.Year() != year || int(date.Month()) != month || date.Day() != day {
		return time.Time{}, NewError(ErrInvalidDate, "invalid calendar date")
	}

	return date, nil
}

// formatEventID formats event ID as string
func formatEventID(num uint64) string {
	return formatID(num, "EVT")
}

// formatCategoryID formats category ID as string
func formatCategoryID(num uint64) string {
	return formatID(num, "CAT")
}

// formatSeatID formats seat ID as string
func formatSeatID(num uint64) string {
	return formatID(num, "SEAT")
}

// formatID formats a numeric ID with a prefix
func formatID(num uint64, prefix string) string {
	return fmt.Sprintf("%s%06d", prefix, num)
}

// IsValidTicketIDFormat validates ticket ID format without decoding
func IsValidTicketIDFormat(ticketID string) bool {
	// Normalize
	ticketID = base32.NormalizeBase32(ticketID)

	// Check length
	if len(ticketID) != TotalLength {
		return false
	}

	// Check all characters are valid Base32
	for _, char := range ticketID {
		if !base32.IsValidBase32Char(char) {
			return false
		}
	}

	// Validate checksum
	return base32.ValidateChecksum(ticketID)
}

// ExtractComponents extracts raw component strings without decoding
func ExtractComponents(ticketID string) map[string]string {
	ticketID = base32.NormalizeBase32(ticketID)

	if len(ticketID) != TotalLength {
		return nil
	}

	return map[string]string{
		"eventID":   ticketID[0:5],
		"eventDate": ticketID[5:10],
		"category":  ticketID[10:13],
		"seatID":    ticketID[13:18],
		"sequence":  ticketID[18:22],
		"checksum":  ticketID[22:24],
	}
}

// RemoveDashes removes all dashes from a ticket ID
func RemoveDashes(ticketID string) string {
	return strings.ReplaceAll(ticketID, "-", "")
}
