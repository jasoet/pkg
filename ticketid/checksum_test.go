package ticketid

import (
	"testing"
	"time"
)

func TestChecksumValidation(t *testing.T) {
	eventDate := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Generate a valid ticket
	validTicket, err := Generate("CHK001", eventDate, "VIP", "A100", 123)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	tests := []struct {
		name     string
		ticketID string
		want     bool
	}{
		{
			name:     "valid checksum",
			ticketID: validTicket,
			want:     true,
		},
		{
			name:     "invalid checksum - last char changed",
			ticketID: validTicket[:23] + "Z",
			want:     false,
		},
		{
			name:     "invalid checksum - second to last char changed",
			ticketID: validTicket[:22] + "Z" + validTicket[23:],
			want:     false,
		},
		{
			name:     "invalid checksum - both checksum chars changed",
			ticketID: validTicket[:22] + "ZZ",
			want:     false,
		},
		{
			name:     "invalid checksum - first char changed",
			ticketID: "Z" + validTicket[1:],
			want:     false,
		},
		{
			name:     "invalid checksum - middle char changed",
			ticketID: validTicket[:12] + "Z" + validTicket[13:],
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidTicketIDFormat(tt.ticketID)
			if got != tt.want {
				t.Errorf("IsValidTicketIDFormat() = %v, want %v for ticket %s", got, tt.want, tt.ticketID)
			}

			// Also test with DecodeTicketID
			_, err := DecodeTicketID(tt.ticketID)
			if tt.want {
				if err != nil {
					t.Errorf("DecodeTicketID() should succeed for valid checksum, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("DecodeTicketID() should fail for invalid checksum")
				}
			}
		})
	}
}

func TestChecksumDetectsAllSingleBitFlips(t *testing.T) {
	eventDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Generate a valid ticket
	validTicket, err := Generate("BIT001", eventDate, "TEST", "SEAT", 999)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Test that modifying any single character is detected
	for i := 0; i < len(validTicket)-2; i++ { // Don't test checksum chars
		modified := []rune(validTicket)
		original := modified[i]

		// Try changing to a different valid base32 character
		if original != '0' {
			modified[i] = '0'
		} else {
			modified[i] = '1'
		}

		modifiedTicket := string(modified)

		if IsValidTicketIDFormat(modifiedTicket) {
			t.Errorf("Checksum failed to detect modification at position %d: %s -> %s", i, validTicket, modifiedTicket)
		}

		_, err := DecodeTicketID(modifiedTicket)
		if err == nil {
			t.Errorf("DecodeTicketID() should fail for modified ticket at position %d", i)
		}
	}
}

func TestChecksumConsistency(t *testing.T) {
	eventDate := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	// Generate same ticket multiple times - should have same checksum
	tickets := make([]string, 10)
	for i := 0; i < 10; i++ {
		ticket, err := Generate("SAME", eventDate, "CAT", "SEAT", 555)
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}
		tickets[i] = ticket
	}

	// All tickets should be identical (including checksum)
	for i := 1; i < len(tickets); i++ {
		if tickets[i] != tickets[0] {
			t.Errorf("Same inputs produced different tickets: %s vs %s", tickets[0], tickets[i])
		}
	}
}

func TestChecksumWithDifferentSequences(t *testing.T) {
	eventDate := time.Date(2025, 4, 10, 0, 0, 0, 0, time.UTC)

	// Generate tickets with different sequences
	checksums := make(map[string]bool)
	tickets := make([]string, 100)

	for i := 0; i < 100; i++ {
		ticket, err := Generate("SEQ", eventDate, "CAT", "SEAT", i)
		if err != nil {
			t.Fatalf("Generate() failed for sequence %d: %v", i, err)
		}
		tickets[i] = ticket

		// Extract checksum
		checksum := ticket[22:24]
		checksums[checksum] = true

		// Validate
		if !IsValidTicketIDFormat(ticket) {
			t.Errorf("Invalid checksum for sequence %d: %s", i, ticket)
		}
	}

	// Should have many different checksums (not all same)
	if len(checksums) < 50 {
		t.Errorf("Too few unique checksums: got %d, want at least 50", len(checksums))
	}
}

func TestChecksumAfterDashRemoval(t *testing.T) {
	eventDate := time.Date(2025, 5, 5, 0, 0, 0, 0, time.UTC)

	ticket, err := Generate("DASH", eventDate, "CAT", "SEAT", 777)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	// Format with dashes
	formatted := Format(ticket)

	// Remove dashes manually
	removed := RemoveDashes(formatted)

	// Should match original
	if removed != ticket {
		t.Errorf("Dash removal mismatch: got %s, want %s", removed, ticket)
	}

	// Both should validate
	if !IsValidTicketIDFormat(ticket) {
		t.Errorf("Original ticket failed validation: %s", ticket)
	}
	if !IsValidTicketIDFormat(formatted) {
		t.Errorf("Formatted ticket failed validation: %s", formatted)
	}
}

func TestChecksumWithMaxValues(t *testing.T) {
	tests := []struct {
		name       string
		eventID    string
		eventDate  time.Time
		categoryID string
		seatID     string
		sequence   int
	}{
		{
			name:       "max event ID",
			eventID:    "33554431",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "1000",
			sequence:   100,
		},
		{
			name:       "max category ID",
			eventID:    "1000",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "32767",
			seatID:     "1000",
			sequence:   100,
		},
		{
			name:       "max seat ID",
			eventID:    "1000",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "33554431",
			sequence:   100,
		},
		{
			name:       "max sequence",
			eventID:    "1000",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "1000",
			sequence:   1048575,
		},
		{
			name:       "max date",
			eventID:    "1000",
			eventDate:  time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "1000",
			sequence:   100,
		},
		{
			name:       "all max values",
			eventID:    "33554431",
			eventDate:  time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
			categoryID: "32767",
			seatID:     "33554431",
			sequence:   1048575,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := Generate(tt.eventID, tt.eventDate, tt.categoryID, tt.seatID, tt.sequence)
			if err != nil {
				t.Fatalf("Generate() failed: %v", err)
			}

			if !IsValidTicketIDFormat(ticket) {
				t.Errorf("Checksum validation failed for %s: %s", tt.name, ticket)
			}

			decoded, err := DecodeTicketID(ticket)
			if err != nil {
				t.Errorf("DecodeTicketID() failed for %s: %v", tt.name, err)
			}

			if decoded.Sequence != tt.sequence {
				t.Errorf("Sequence mismatch for %s: got %d, want %d", tt.name, decoded.Sequence, tt.sequence)
			}
		})
	}
}

func TestChecksumWithMinValues(t *testing.T) {
	tests := []struct {
		name       string
		eventID    string
		eventDate  time.Time
		categoryID string
		seatID     string
		sequence   int
	}{
		{
			name:       "min numeric values",
			eventID:    "0",
			eventDate:  time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "0",
			seatID:     "0",
			sequence:   0,
		},
		{
			name:       "min event ID",
			eventID:    "0",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "1000",
			sequence:   100,
		},
		{
			name:       "min category ID",
			eventID:    "1000",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "0",
			seatID:     "1000",
			sequence:   100,
		},
		{
			name:       "min seat ID",
			eventID:    "1000",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "0",
			sequence:   100,
		},
		{
			name:       "min sequence",
			eventID:    "1000",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "1000",
			sequence:   0,
		},
		{
			name:       "min date",
			eventID:    "1000",
			eventDate:  time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "1000",
			sequence:   100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := Generate(tt.eventID, tt.eventDate, tt.categoryID, tt.seatID, tt.sequence)
			if err != nil {
				t.Fatalf("Generate() failed: %v", err)
			}

			if !IsValidTicketIDFormat(ticket) {
				t.Errorf("Checksum validation failed for %s: %s", tt.name, ticket)
			}

			decoded, err := DecodeTicketID(ticket)
			if err != nil {
				t.Errorf("DecodeTicketID() failed for %s: %v", tt.name, err)
			}

			if decoded.Sequence != tt.sequence {
				t.Errorf("Sequence mismatch for %s: got %d, want %d", tt.name, decoded.Sequence, tt.sequence)
			}
		})
	}
}
