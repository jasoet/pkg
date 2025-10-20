package ticketid

import (
	"strings"
	"testing"
	"time"
)

func TestErrorTypes(t *testing.T) {
	tests := []struct {
		name         string
		generateFunc func() error
		wantCode     string
	}{
		{
			name: "ErrInvalidEventID - empty",
			generateFunc: func() error {
				_, err := Generate("", time.Now(), "CAT", "SEAT", 1)
				return err
			},
			wantCode: ErrInvalidEventID,
		},
		{
			name: "ErrInvalidCategory - empty",
			generateFunc: func() error {
				_, err := Generate("EVT", time.Now(), "", "SEAT", 1)
				return err
			},
			wantCode: ErrInvalidCategory,
		},
		{
			name: "ErrInvalidSeat - empty",
			generateFunc: func() error {
				_, err := Generate("EVT", time.Now(), "CAT", "", 1)
				return err
			},
			wantCode: ErrInvalidSeat,
		},
		{
			name: "ErrInvalidSequence - negative",
			generateFunc: func() error {
				_, err := Generate("EVT", time.Now(), "CAT", "SEAT", -1)
				return err
			},
			wantCode: ErrInvalidSequence,
		},
		{
			name: "ErrValueOutOfRange - event ID",
			generateFunc: func() error {
				_, err := Generate("33554432", time.Now(), "CAT", "SEAT", 1)
				return err
			},
			wantCode: ErrValueOutOfRange,
		},
		{
			name: "ErrValueOutOfRange - category ID",
			generateFunc: func() error {
				_, err := Generate("EVT", time.Now(), "32768", "SEAT", 1)
				return err
			},
			wantCode: ErrValueOutOfRange,
		},
		{
			name: "ErrValueOutOfRange - seat ID",
			generateFunc: func() error {
				_, err := Generate("EVT", time.Now(), "CAT", "33554432", 1)
				return err
			},
			wantCode: ErrValueOutOfRange,
		},
		{
			name: "ErrInvalidLength - decode too short",
			generateFunc: func() error {
				_, err := DecodeTicketID("SHORT")
				return err
			},
			wantCode: ErrInvalidLength,
		},
		{
			name: "ErrInvalidChecksum - wrong checksum",
			generateFunc: func() error {
				validTicket, _ := Generate("EVT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "CAT", "SEAT", 1)
				invalidTicket := validTicket[:22] + "ZZ"
				_, err := DecodeTicketID(invalidTicket)
				return err
			},
			wantCode: ErrInvalidChecksum,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.generateFunc()
			if err == nil {
				t.Fatalf("Expected error, got nil")
			}

			// Check if error contains expected code
			if ticketErr, ok := err.(*Error); ok {
				if ticketErr.Code != tt.wantCode {
					t.Errorf("Error code = %v, want %v", ticketErr.Code, tt.wantCode)
				}
			} else {
				// For wrapped errors, check the error message
				if !strings.Contains(err.Error(), tt.wantCode) {
					t.Errorf("Error message should contain code %v, got: %v", tt.wantCode, err.Error())
				}
			}
		})
	}
}

func TestErrorMethods(t *testing.T) {
	t.Run("Error.Error() without details", func(t *testing.T) {
		err := NewError("TEST_CODE", "test message")
		errStr := err.Error()
		if !strings.Contains(errStr, "TEST_CODE") {
			t.Errorf("Error string should contain code, got: %s", errStr)
		}
		if !strings.Contains(errStr, "test message") {
			t.Errorf("Error string should contain message, got: %s", errStr)
		}
	})

	t.Run("Error.Error() with details", func(t *testing.T) {
		details := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}
		err := NewErrorWithDetails("TEST_CODE", "test message", details)
		errStr := err.Error()
		if !strings.Contains(errStr, "TEST_CODE") {
			t.Errorf("Error string should contain code, got: %s", errStr)
		}
		if !strings.Contains(errStr, "test message") {
			t.Errorf("Error string should contain message, got: %s", errStr)
		}
		if !strings.Contains(errStr, "details") {
			t.Errorf("Error string should contain details, got: %s", errStr)
		}
	})

	t.Run("WrapError with nil error", func(t *testing.T) {
		err := WrapError("TEST_CODE", "test message", nil)
		if ticketErr, ok := err.(*Error); !ok {
			t.Errorf("WrapError with nil should return *Error, got %T", err)
		} else if ticketErr.Code != "TEST_CODE" {
			t.Errorf("WrapError code = %v, want TEST_CODE", ticketErr.Code)
		}
	})

	t.Run("WrapError with existing error", func(t *testing.T) {
		baseErr := NewError("BASE_CODE", "base message")
		wrapped := WrapError("WRAP_CODE", "wrap message", baseErr)
		errStr := wrapped.Error()
		if !strings.Contains(errStr, "WRAP_CODE") {
			t.Errorf("Wrapped error should contain wrap code, got: %s", errStr)
		}
		if !strings.Contains(errStr, "wrap message") {
			t.Errorf("Wrapped error should contain wrap message, got: %s", errStr)
		}
	})
}

func TestEdgeCaseDates(t *testing.T) {
	tests := []struct {
		name      string
		dateNum   uint64
		wantErr   bool
		errorCode string
	}{
		{
			name:    "valid leap year - Feb 29, 2024",
			dateNum: 20240229,
			wantErr: false,
		},
		{
			name:      "invalid non-leap year - Feb 29, 2025",
			dateNum:   20250229,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "invalid date - Feb 30",
			dateNum:   20250230,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "invalid date - Apr 31",
			dateNum:   20250431,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:    "valid date - Apr 30",
			dateNum: 20250430,
			wantErr: false,
		},
		{
			name:      "year too old - 1969",
			dateNum:   19690101,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "year too far - 2101",
			dateNum:   21010101,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "month 0",
			dateNum:   20250001,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "month 13",
			dateNum:   20251301,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "day 0",
			dateNum:   20250100,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
		{
			name:      "day 32",
			dateNum:   20250132,
			wantErr:   true,
			errorCode: ErrInvalidDate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDateFromEncoded(tt.dateNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateFromEncoded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				if ticketErr, ok := err.(*Error); ok {
					if ticketErr.Code != tt.errorCode {
						t.Errorf("Error code = %v, want %v", ticketErr.Code, tt.errorCode)
					}
				}
			}
		})
	}
}

func TestEdgeCaseInputs(t *testing.T) {
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		eventID    string
		categoryID string
		seatID     string
		sequence   int
		wantErr    bool
	}{
		{
			name:       "very long event ID string",
			eventID:    strings.Repeat("A", 100),
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false, // Should hash it
		},
		{
			name:       "very long category ID string",
			eventID:    "EVT",
			categoryID: strings.Repeat("B", 100),
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false, // Should hash it
		},
		{
			name:       "very long seat ID string",
			eventID:    "EVT",
			categoryID: "CAT",
			seatID:     strings.Repeat("C", 100),
			sequence:   1,
			wantErr:    false, // Should hash it
		},
		{
			name:       "special characters in event ID",
			eventID:    "EVT-2025!@#",
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false, // Should hash it
		},
		{
			name:       "unicode in category ID",
			eventID:    "EVT",
			categoryID: "КАТЕГОРИЯ",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false, // Should hash it
		},
		{
			name:       "whitespace in seat ID",
			eventID:    "EVT",
			categoryID: "CAT",
			seatID:     "SEAT 123",
			sequence:   1,
			wantErr:    false, // Should hash it
		},
		{
			name:       "single character IDs",
			eventID:    "A",
			categoryID: "B",
			seatID:     "C",
			sequence:   1,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := Generate(tt.eventID, validDate, tt.categoryID, tt.seatID, tt.sequence)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify the ticket can be decoded
				decoded, err := DecodeTicketID(ticket)
				if err != nil {
					t.Errorf("DecodeTicketID() failed for generated ticket: %v", err)
				}
				if decoded.Sequence != tt.sequence {
					t.Errorf("Sequence mismatch: got %d, want %d", decoded.Sequence, tt.sequence)
				}
			}
		})
	}
}

func TestDecodeEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		ticketID  string
		wantErr   bool
		errorCode string
		setupFunc func() string
	}{
		{
			name:      "empty string",
			ticketID:  "",
			wantErr:   true,
			errorCode: ErrInvalidLength,
		},
		{
			name:      "too short",
			ticketID:  "ABC",
			wantErr:   true,
			errorCode: ErrInvalidLength,
		},
		{
			name:      "too long",
			ticketID:  "ABCDEFGHIJKLMNOPQRSTUVWXYZ",
			wantErr:   true,
			errorCode: ErrInvalidLength,
		},
		{
			name: "exactly 24 chars but invalid checksum",
			setupFunc: func() string {
				// Generate valid ticket, then change checksum to make it invalid
				ticket, _ := Generate("EVT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "CAT", "SEAT", 1)
				return ticket[:22] + "00"
			},
			wantErr:   true,
			errorCode: ErrInvalidChecksum,
		},
		{
			name:      "with spaces",
			ticketID:  "01B2M 4K6G8 N3V 9F2HA 7XJ5 QR",
			wantErr:   true,
			errorCode: ErrInvalidChecksum, // Spaces are removed by normalization, making checksum invalid
		},
		{
			name:     "lowercase valid ticket",
			ticketID: func() string { ticket, _ := Generate("EVT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "CAT", "SEAT", 1); return strings.ToLower(ticket) }(),
			wantErr:  false,
		},
		{
			name:     "mixed case valid ticket",
			ticketID: func() string { ticket, _ := Generate("EVT", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), "CAT", "SEAT", 1); return strings.ToLower(ticket[:12]) + strings.ToUpper(ticket[12:]) }(),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketID := tt.ticketID
			if tt.setupFunc != nil {
				ticketID = tt.setupFunc()
			}

			_, err := DecodeTicketID(ticketID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeTicketID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errorCode != "" {
				if ticketErr, ok := err.(*Error); ok {
					if ticketErr.Code != tt.errorCode {
						t.Errorf("Error code = %v, want %v", ticketErr.Code, tt.errorCode)
					}
				} else {
					// For wrapped errors, check error message
					if !strings.Contains(err.Error(), tt.errorCode) {
						t.Errorf("Error should contain code %v, got: %v", tt.errorCode, err.Error())
					}
				}
			}
		})
	}
}

func TestSequenceBoundaries(t *testing.T) {
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		sequence int
		wantErr  bool
	}{
		{
			name:     "sequence 0",
			sequence: 0,
			wantErr:  false,
		},
		{
			name:     "sequence 1",
			sequence: 1,
			wantErr:  false,
		},
		{
			name:     "sequence max-1",
			sequence: 1048574,
			wantErr:  false,
		},
		{
			name:     "sequence max",
			sequence: 1048575,
			wantErr:  false,
		},
		{
			name:     "sequence over max (should wrap)",
			sequence: 1048576,
			wantErr:  false,
		},
		{
			name:     "sequence negative",
			sequence: -1,
			wantErr:  true,
		},
		{
			name:     "sequence very negative",
			sequence: -1000000,
			wantErr:  true,
		},
		{
			name:     "sequence very large",
			sequence: 10000000,
			wantErr:  false, // Should wrap
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticket, err := Generate("EVT", validDate, "CAT", "SEAT", tt.sequence)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Decode and verify
				decoded, err := DecodeTicketID(ticket)
				if err != nil {
					t.Errorf("DecodeTicketID() failed: %v", err)
					return
				}
				// For sequences over max, they should wrap
				expectedSeq := tt.sequence
				if expectedSeq >= 1048576 {
					expectedSeq = expectedSeq % 1048576
				}
				if decoded.Sequence != expectedSeq {
					t.Errorf("Sequence mismatch: got %d, want %d", decoded.Sequence, expectedSeq)
				}
			}
		})
	}
}

func TestFormatEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		ticketID string
		want     string
	}{
		{
			name:     "empty string",
			ticketID: "",
			want:     "",
		},
		{
			name:     "too short",
			ticketID: "SHORT",
			want:     "SHORT",
		},
		{
			name:     "too long",
			ticketID: "TOOLONGTICKETIDWITHEXTRACHARACTERS",
			want:     "TOOLONGTICKETIDWITHEXTRACHARACTERS",
		},
		{
			name:     "exactly 24 chars",
			ticketID: "012345678901234567890123",
			want:     "01234-56789-012-34567-8901-23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Format(tt.ticketID); got != tt.want {
				t.Errorf("Format() = %v, want %v", got, tt.want)
			}
		})
	}
}
