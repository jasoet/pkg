package ticketid

import "time"

// TicketID represents the decoded components of a ticket ID
type TicketID struct {
	EventID    string    `json:"eventId" msgpack:"e"`
	EventDate  time.Time `json:"eventDate" msgpack:"d"`
	CategoryID string    `json:"categoryId" msgpack:"c"`
	SeatID     string    `json:"seatId" msgpack:"s"`
	Sequence   int       `json:"sequence" msgpack:"q"`
	EncodedID  string    `json:"encodedId" msgpack:"i"`
	Checksum   string    `json:"checksum" msgpack:"k"`
}

// Ticket ID component lengths (in Base32 characters)
const (
	EventIDLength   = 5 // 33.5M unique events
	EventDateLength = 5 // YYYYMMDD encoded
	CategoryLength  = 3 // 32,768 categories
	SeatIDLength    = 5 // 33.5M seats
	SequenceLength  = 4 // 1M sequences
	ChecksumLength  = 2 // 2-char CRC-10

	// TotalLength: 5+5+3+5+4+2 = 24 characters
	TotalLength = EventIDLength + EventDateLength + CategoryLength + SeatIDLength + SequenceLength + ChecksumLength
)
