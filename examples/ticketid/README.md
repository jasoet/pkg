# TicketID Package Examples

This directory contains examples demonstrating how to use the `ticketid` package for generating and decoding 24-character ticket IDs with Base32 encoding and CRC-10 checksum validation.

## 📍 Example Code Location

**Full example implementation:** [/ticketid/examples/example.go](https://github.com/jasoet/pkg/blob/main/ticketid/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/v2/ticketid"

// Generate a ticket ID
ticketID, err := ticketid.Generate(
    "EVT001",                      // Event ID (numeric or string)
    time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC), // Event Date
    "VIP",                         // Category ID
    "A123",                        // Seat ID
    ticketid.GenerateSequence(),   // Auto-generated unique sequence
)

// Decode a ticket ID
decoded, err := ticketid.DecodeTicketID(ticketID)
fmt.Printf("Event: %s, Date: %s, Category: %s\n",
    decoded.EventID,
    decoded.EventDate.Format("2006-01-02"),
    decoded.CategoryID)

// Validate ticket format and checksum
if ticketid.IsValidTicketIDFormat(ticketID) {
    fmt.Println("Valid ticket!")
}

// Format for display
formatted := ticketid.Format(ticketID)  // "01B2M-4K6G8-N3V-9F2HA-7XJ5-QR"
```

**Key Features:**
- Compact 24-character IDs with 5 encoded components
- 99.9% error detection via CRC-10 checksum
- Human-readable Base32 encoding (URL-safe)
- Supports both numeric and string input IDs
- Date range: 1970-2100

## Overview

The `ticketid` package provides a robust solution for generating and managing ticket IDs with:
- **Event ID encoding** (5 chars, 33.5M unique events)
- **Event Date encoding** (5 chars, YYYYMMDD format)
- **Category encoding** (3 chars, 32,768 categories)
- **Seat ID encoding** (5 chars, 33.5M seats)
- **Sequence number** (4 chars, 1M sequences)
- **CRC-10 Checksum** (2 chars for error detection)

## Running the Examples

To run the examples, use the following command from the `ticketid/examples` directory:

```bash
go run example.go
```

Or from the repository root:

```bash
go run ./examples/ticketid
```

This will demonstrate:
1. Basic ticket generation with numeric and string IDs
2. Decoding and validation of ticket IDs
3. Complete event ticketing system scenario
4. Comprehensive error handling
5. Formatting and component extraction
6. Batch ticket generation

## Example Descriptions

The [example.go](https://github.com/jasoet/pkg/blob/main/ticketid/examples/example.go) file demonstrates several use cases:

### 1. Basic Generation

Shows how to create ticket IDs with both numeric and string inputs:

```go
// Numeric IDs
ticket1, err := ticketid.Generate("12345", eventDate, "100", "500", sequence)

// String IDs
ticket2, err := ticketid.Generate("EVT001", eventDate, "VIP", "A123", sequence)
```

### 2. Decoding and Validation

Demonstrates decoding ticket IDs and accessing components:

```go
decoded, err := ticketid.DecodeTicketID(ticketID)
fmt.Printf("Event ID:    %s\n", decoded.EventID)
fmt.Printf("Event Date:  %s\n", decoded.EventDate.Format("2006-01-02"))
fmt.Printf("Category:    %s\n", decoded.CategoryID)
fmt.Printf("Seat ID:     %s\n", decoded.SeatID)
fmt.Printf("Sequence:    %d\n", decoded.Sequence)
```

### 3. Event Ticketing System

Complete scenario showing ticket generation for different categories:

```go
// VIP tickets
vipTicket, _ := ticketid.Generate(eventName, eventDate, "VIP", "VIP-A1", 1001)

// Premium tickets
premiumTicket, _ := ticketid.Generate(eventName, eventDate, "PREMIUM", "PREM-B10", 2001)

// General admission
generalTicket, _ := ticketid.Generate(eventName, eventDate, "GENERAL", "GEN-C50", 3001)
```

### 4. Error Handling

Comprehensive error handling for common scenarios:

```go
// Empty event ID
_, err := ticketid.Generate("", time.Now(), "CAT", "SEAT", 1)

// Negative sequence
_, err = ticketid.Generate("EVT", time.Now(), "CAT", "SEAT", -1)

// Value out of range
_, err = ticketid.Generate("33554432", time.Now(), "CAT", "SEAT", 1)

// Invalid checksum
_, err = ticketid.DecodeTicketID(corruptedTicket)
```

### 5. Formatting and Components

Shows formatting options and component extraction:

```go
// Format with dashes for readability
formatted := ticketid.Format(ticket)  // "01B2M-4K6G8-N3V-9F2HA-7XJ5-QR"

// Extract raw components without full decoding
components := ticketid.ExtractComponents(ticket)
fmt.Printf("Event ID: %s\n", components["eventID"])

// Remove dashes
withoutDashes := ticketid.RemoveDashes(formatted)
```

### 6. Batch Generation

Demonstrates generating multiple unique tickets:

```go
for i := 0; i < 10; i++ {
    ticket, err := ticketid.Generate(
        eventName,
        eventDate,
        "GA",
        fmt.Sprintf("SEAT-%03d", i+1),
        ticketid.GenerateSequence(),  // Unique sequence for each
    )
}
```

## Ticket ID Format

Each 24-character ticket ID encodes the following information:

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

**Example:** `01B2M4K6G8N3V9F2HA7XJ5QR`

**Formatted:** `01B2M-4K6G8-N3V-9F2HA-7XJ5-QR`

## Security and Error Detection

### Checksum Validation

The CRC-10 checksum provides excellent error detection:

| Error Type | Detection Rate |
|------------|----------------|
| Single character error | 100% |
| Transposition (AB→BA) | 99.9%+ |
| Double errors | 99.9%+ |
| Insertion/deletion | High |

### Input Validation

The package validates all inputs:
- Event IDs, categories, and seat IDs cannot be empty
- Sequence must be non-negative
- Numeric values must be within valid ranges
- Dates must be between 1970-2100

### Error Codes

All errors include specific error codes:
- `ErrInvalidEventID` - Event ID is invalid or empty
- `ErrInvalidDate` - Date is out of range or invalid
- `ErrInvalidCategory` - Category ID is invalid
- `ErrInvalidSeat` - Seat ID is invalid
- `ErrInvalidSequence` - Sequence number is invalid
- `ErrInvalidChecksum` - Checksum validation failed
- `ErrInvalidLength` - Ticket ID length is incorrect
- `ErrValueOutOfRange` - Numeric value exceeds limit

## Use Cases

### Event Ticketing

Generate unique, verifiable tickets for events with multiple categories and seating arrangements.

### Access Control

Create short, human-readable access codes that can be validated offline.

### Order Tracking

Encode order information (date, category, sequence) into compact IDs.

### Voucher Systems

Generate unique voucher codes with built-in error detection.

## Key Features

- **Compact**: Only 24 characters encode all information
- **Error Detection**: 99.9% error detection via CRC-10 checksum
- **Human-Friendly**: Base32 encoding is readable and URL-safe
- **Flexible**: Accepts both numeric and string inputs
- **Type-Safe**: Structured decoding with validation
- **Self-Contained**: No external dependencies except base32 package

## Dependencies

- `github.com/jasoet/pkg/v2/base32` - Base32 encoding with checksum support

## Further Reading

- [TicketID Package Documentation](https://github.com/jasoet/pkg/tree/main/ticketid)
- [Base32 Package Documentation](https://github.com/jasoet/pkg/tree/main/base32)
- [Main pkg Repository](https://github.com/jasoet/pkg)
