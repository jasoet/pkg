# TicketID

A reusable Go package for generating and decoding ticket IDs with Base32 encoding and CRC-10 checksum validation.

## Features

- **Compact IDs**: 24-character ticket IDs encoding multiple components
- **Checksum Validation**: CRC-10 checksum provides 99.9% error detection
- **Base32 Encoding**: Human-readable and URL-safe
- **Type-Safe**: Structured decoding with validation
- **Flexible Input**: Supports both numeric and string inputs for IDs

## Installation

```bash
go get github.com/jasoet/pkg/v2/ticketid
```

## Format

Ticket IDs are 24 characters long with the following structure:

```
[EEEEE][DDDDD][CCC][SSSSS][RRRR][XX]
   5      5     3     5      4    2  = 24 characters

EEEEE - Event ID (5 chars, 33.5M unique events)
DDDDD - Event Date (5 chars, YYYYMMDD encoded)
CCC   - Category (3 chars, 32,768 categories)
SSSSS - Seat ID (5 chars, 33.5M seats)
RRRR  - Sequence (4 chars, 1M sequences)
XX    - CRC-10 Checksum (2 chars)
```

**Example**: `01B2M4K6G8N3V9F2HA7XJ5QR`

## Usage

### Generate a Ticket ID

```go
package main

import (
    "fmt"
    "time"

    "github.com/jasoet/pkg/v2/ticketid"
)

func main() {
    // Generate with explicit sequence
    ticketID, err := ticketid.Generate(
        "EVT001",                      // Event ID
        time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Event Date
        "VIP",                         // Category
        "A123",                        // Seat ID
        ticketid.GenerateSequence(),   // Auto-generated sequence
    )
    if err != nil {
        panic(err)
    }

    fmt.Println("Ticket ID:", ticketID)
    fmt.Println("Formatted:", ticketid.Format(ticketID))
    // Output: 01B2M-4K6G8-N3V-9F2HA-7XJ5-QR
}
```

### Decode a Ticket ID

```go
decoded, err := ticketid.DecodeTicketID("01B2M4K6G8N3V9F2HA7XJ5QR")
if err != nil {
    panic(err)
}

fmt.Printf("Event ID:    %s\n", decoded.EventID)
fmt.Printf("Event Date:  %s\n", decoded.EventDate.Format("2006-01-02"))
fmt.Printf("Category:    %s\n", decoded.CategoryID)
fmt.Printf("Seat ID:     %s\n", decoded.SeatID)
fmt.Printf("Sequence:    %d\n", decoded.Sequence)
```

### Validate Ticket ID Format

```go
if ticketid.IsValidTicketIDFormat("01B2M4K6G8N3V9F2HA7XJ5QR") {
    fmt.Println("Valid ticket ID")
}
```

## API Reference

### Functions

- `Generate(eventID, date, categoryID, seatID, sequence)` - Generate a new ticket ID
- `GenerateSequence()` - Generate a random sequence number
- `DecodeTicketID(ticketID)` - Decode ticket ID into components
- `ParseTicketID(ticketID)` - Alias for DecodeTicketID
- `Format(ticketID)` - Format with dashes for readability
- `IsValidTicketIDFormat(ticketID)` - Validate format and checksum
- `ExtractComponents(ticketID)` - Extract raw component strings
- `RemoveDashes(ticketID)` - Remove formatting dashes

### Types

```go
type TicketID struct {
    EventID    string
    EventDate  time.Time
    CategoryID string
    SeatID     string
    Sequence   int
    EncodedID  string
    Checksum   string
}

type Error struct {
    Code    string
    Message string
    Details map[string]interface{}
}
```

### Error Codes

- `ErrInvalidEventID` - Event ID is invalid or empty
- `ErrInvalidDate` - Date is out of range or invalid
- `ErrInvalidCategory` - Category ID is invalid
- `ErrInvalidSeat` - Seat ID is invalid
- `ErrInvalidSequence` - Sequence number is invalid
- `ErrInvalidTicketID` - Ticket ID format is invalid
- `ErrDecodeFailed` - Failed to decode component
- `ErrInvalidChecksum` - Checksum validation failed
- `ErrInvalidLength` - Ticket ID length is incorrect
- `ErrValueOutOfRange` - Numeric value exceeds limit

## Dependencies

- `github.com/jasoet/pkg/v2/base32` - Base32 encoding with checksum support

## License

See the LICENSE file in the parent repository.
