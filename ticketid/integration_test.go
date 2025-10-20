package ticketid

import (
	"testing"
	"time"
)

func TestRoundTripEncoding(t *testing.T) {
	tests := []struct {
		name       string
		eventID    string
		eventDate  time.Time
		categoryID string
		seatID     string
		sequence   int
	}{
		{
			name:       "numeric IDs",
			eventID:    "12345",
			eventDate:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "500",
			sequence:   999,
		},
		{
			name:       "string IDs",
			eventID:    "EVT001",
			eventDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			categoryID: "VIP",
			seatID:     "A123",
			sequence:   12345,
		},
		{
			name:       "mixed IDs",
			eventID:    "42",
			eventDate:  time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
			categoryID: "GENERAL",
			seatID:     "B456",
			sequence:   0,
		},
		{
			name:       "edge case - max numeric values",
			eventID:    "33554431",
			eventDate:  time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
			categoryID: "32767",
			seatID:     "33554431",
			sequence:   1048575,
		},
		{
			name:       "edge case - min values",
			eventID:    "0",
			eventDate:  time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "0",
			seatID:     "0",
			sequence:   0,
		},
		{
			name:       "special characters in string IDs",
			eventID:    "EVENT-2025",
			eventDate:  time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
			categoryID: "VIP-GOLD",
			seatID:     "SEAT_A1",
			sequence:   777,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Step 1: Generate ticket ID
			ticketID, err := Generate(tt.eventID, tt.eventDate, tt.categoryID, tt.seatID, tt.sequence)
			if err != nil {
				t.Fatalf("Generate() failed: %v", err)
			}

			// Verify generated ticket ID is valid format
			if !IsValidTicketIDFormat(ticketID) {
				t.Errorf("Generated ticket ID has invalid format: %s", ticketID)
			}

			// Step 2: Decode the ticket ID
			decoded, err := DecodeTicketID(ticketID)
			if err != nil {
				t.Fatalf("DecodeTicketID() failed: %v", err)
			}

			// Step 3: Verify decoded values
			// Note: String IDs are hashed, so we can only verify numeric IDs match exactly
			if decoded.Sequence != tt.sequence {
				t.Errorf("Sequence mismatch: got %d, want %d", decoded.Sequence, tt.sequence)
			}

			// Verify date matches (only date part, not time)
			if !decoded.EventDate.Equal(tt.eventDate) {
				t.Errorf("EventDate mismatch: got %v, want %v", decoded.EventDate, tt.eventDate)
			}

			// Verify encoded ID is set
			if decoded.EncodedID != ticketID {
				t.Errorf("EncodedID mismatch: got %s, want %s", decoded.EncodedID, ticketID)
			}

			// Step 4: Verify ticket ID length
			if len(ticketID) != TotalLength {
				t.Errorf("Ticket ID length: got %d, want %d", len(ticketID), TotalLength)
			}
		})
	}
}

func TestRoundTripWithFormatting(t *testing.T) {
	eventDate := time.Date(2025, 10, 20, 0, 0, 0, 0, time.UTC)

	// Generate ticket
	ticketID, err := Generate("TEST123", eventDate, "PREMIUM", "SEAT777", 456)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Format with dashes
	formatted := Format(ticketID)

	// Decode formatted ticket ID (should handle dashes)
	decoded, err := DecodeTicketID(formatted)
	if err != nil {
		t.Fatalf("DecodeTicketID() failed with formatted input: %v", err)
	}

	// Verify decoded values
	if !decoded.EventDate.Equal(eventDate) {
		t.Errorf("EventDate mismatch after formatting: got %v, want %v", decoded.EventDate, eventDate)
	}

	if decoded.Sequence != 456 {
		t.Errorf("Sequence mismatch after formatting: got %d, want %d", decoded.Sequence, 456)
	}

	// Verify encoded ID matches original (without dashes)
	if decoded.EncodedID != ticketID {
		t.Errorf("EncodedID mismatch: got %s, want %s", decoded.EncodedID, ticketID)
	}
}

func TestMultipleTicketsUniqueness(t *testing.T) {
	eventDate := time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC)
	generated := make(map[string]bool)

	// Generate 100 tickets with different sequences
	for i := 0; i < 100; i++ {
		ticketID, err := Generate("EVENT1", eventDate, "CAT1", "SEAT1", i)
		if err != nil {
			t.Fatalf("Generate() failed at iteration %d: %v", i, err)
		}

		// Check for duplicates
		if generated[ticketID] {
			t.Errorf("Duplicate ticket ID generated: %s", ticketID)
		}
		generated[ticketID] = true

		// Verify each ticket can be decoded
		decoded, err := DecodeTicketID(ticketID)
		if err != nil {
			t.Errorf("DecodeTicketID() failed for ticket %d: %v", i, err)
		}

		// Verify sequence matches
		if decoded.Sequence != i {
			t.Errorf("Sequence mismatch for ticket %d: got %d, want %d", i, decoded.Sequence, i)
		}
	}

	// Verify we generated 100 unique tickets
	if len(generated) != 100 {
		t.Errorf("Expected 100 unique tickets, got %d", len(generated))
	}
}

func TestRoundTripWithRandomSequences(t *testing.T) {
	eventDate := time.Date(2025, 7, 4, 0, 0, 0, 0, time.UTC)

	// Generate 50 tickets with random sequences
	for i := 0; i < 50; i++ {
		sequence := GenerateSequence()

		ticketID, err := Generate("RANDOM", eventDate, "TEST", "SEAT", sequence)
		if err != nil {
			t.Fatalf("Generate() failed with random sequence: %v", err)
		}

		decoded, err := DecodeTicketID(ticketID)
		if err != nil {
			t.Fatalf("DecodeTicketID() failed: %v", err)
		}

		// Verify sequence is preserved
		if decoded.Sequence != sequence {
			t.Errorf("Sequence mismatch: got %d, want %d", decoded.Sequence, sequence)
		}
	}
}

func TestConcurrentGenerationAndDecoding(t *testing.T) {
	eventDate := time.Date(2025, 8, 15, 0, 0, 0, 0, time.UTC)
	done := make(chan bool)
	errors := make(chan error, 10)

	// Run 10 concurrent goroutines
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			// Each goroutine generates and decodes 10 tickets
			for j := 0; j < 10; j++ {
				sequence := id*10 + j

				ticketID, err := Generate("CONCURRENT", eventDate, "TEST", "SEAT", sequence)
				if err != nil {
					errors <- err
					return
				}

				decoded, err := DecodeTicketID(ticketID)
				if err != nil {
					errors <- err
					return
				}

				if decoded.Sequence != sequence {
					errors <- err
					return
				}
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Check for errors
	close(errors)
	for err := range errors {
		t.Errorf("Concurrent test error: %v", err)
	}
}

func TestExtractAndDecode(t *testing.T) {
	eventDate := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)

	ticketID, err := Generate("EXTRACT", eventDate, "CAT", "SEAT", 555)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Extract components
	components := ExtractComponents(ticketID)
	if components == nil {
		t.Fatal("ExtractComponents() returned nil")
	}

	// Decode full ticket
	decoded, err := DecodeTicketID(ticketID)
	if err != nil {
		t.Fatalf("DecodeTicketID() failed: %v", err)
	}

	// Verify checksum matches
	if components["checksum"] != decoded.Checksum {
		t.Errorf("Checksum mismatch: extracted %s, decoded %s", components["checksum"], decoded.Checksum)
	}

	// Verify we can reconstruct the ticket ID from components
	reconstructed := components["eventID"] + components["eventDate"] + components["category"] + components["seatID"] + components["sequence"] + components["checksum"]
	if reconstructed != ticketID {
		t.Errorf("Reconstructed ticket ID mismatch: got %s, want %s", reconstructed, ticketID)
	}
}

func TestVariousDateRanges(t *testing.T) {
	dates := []time.Time{
		time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1999, 12, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC), // Leap year
		time.Date(2025, 10, 20, 0, 0, 0, 0, time.UTC), // Today
		time.Date(2050, 6, 15, 0, 0, 0, 0, time.UTC),
		time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
	}

	for _, date := range dates {
		t.Run(date.Format("2006-01-02"), func(t *testing.T) {
			ticketID, err := Generate("DATE", date, "TEST", "SEAT", 1)
			if err != nil {
				t.Fatalf("Generate() failed for date %v: %v", date, err)
			}

			decoded, err := DecodeTicketID(ticketID)
			if err != nil {
				t.Fatalf("DecodeTicketID() failed for date %v: %v", date, err)
			}

			if !decoded.EventDate.Equal(date) {
				t.Errorf("Date mismatch: got %v, want %v", decoded.EventDate, date)
			}
		})
	}
}
