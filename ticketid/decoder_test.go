package ticketid

import (
	"testing"
	"time"

	"github.com/jasoet/pkg/v2/base32"
)

func TestDecodeTicketID(t *testing.T) {
	// First, generate valid ticket IDs for testing
	validDate := time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC)
	validTicket, err := Generate("12345", validDate, "VIP", "A123", 999)
	if err != nil {
		t.Fatalf("Failed to generate valid ticket for testing: %v", err)
	}

	tests := []struct {
		name      string
		ticketID  string
		wantErr   bool
		setupFunc func() string // Function to setup ticket ID
	}{
		{
			name:     "valid ticket ID",
			ticketID: validTicket,
			wantErr:  false,
		},
		{
			name:     "valid ticket ID with dashes",
			ticketID: Format(validTicket),
			wantErr:  false,
		},
		{
			name:     "lowercase ticket ID",
			ticketID: func() string { return base32.NormalizeBase32(validTicket) }(),
			wantErr:  false,
		},
		{
			name:     "wrong length - too short",
			ticketID: "SHORT",
			wantErr:  true,
		},
		{
			name:     "wrong length - too long",
			ticketID: "TOOLONGTICKETIDWITHEXTRACHARACTERS",
			wantErr:  true,
		},
		{
			name:     "empty ticket ID",
			ticketID: "",
			wantErr:  true,
		},
		{
			name:     "invalid checksum",
			ticketID: func() string { return validTicket[:22] + "ZZ" }(),
			wantErr:  true,
		},
		{
			name: "another valid ticket",
			setupFunc: func() string {
				ticket, _ := Generate("EVT001", time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC), "100", "SEAT500", 12345)
				return ticket
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketID := tt.ticketID
			if tt.setupFunc != nil {
				ticketID = tt.setupFunc()
			}

			got, err := DecodeTicketID(ticketID)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeTicketID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got == nil {
					t.Errorf("DecodeTicketID() returned nil for valid ticket")
					return
				}
				// Verify all fields are populated
				if got.EventID == "" {
					t.Errorf("DecodeTicketID() EventID is empty")
				}
				if got.EventDate.IsZero() {
					t.Errorf("DecodeTicketID() EventDate is zero")
				}
				if got.CategoryID == "" {
					t.Errorf("DecodeTicketID() CategoryID is empty")
				}
				if got.SeatID == "" {
					t.Errorf("DecodeTicketID() SeatID is empty")
				}
				if got.Sequence < 0 {
					t.Errorf("DecodeTicketID() Sequence is negative: %d", got.Sequence)
				}
				if got.EncodedID == "" {
					t.Errorf("DecodeTicketID() EncodedID is empty")
				}
				if got.Checksum == "" {
					t.Errorf("DecodeTicketID() Checksum is empty")
				}
			}
		})
	}
}

func TestParseTicketID(t *testing.T) {
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validTicket, _ := Generate("TEST", validDate, "CAT", "SEAT", 100)

	// ParseTicketID should behave exactly like DecodeTicketID
	decoded, err1 := DecodeTicketID(validTicket)
	parsed, err2 := ParseTicketID(validTicket)

	if (err1 == nil) != (err2 == nil) {
		t.Errorf("ParseTicketID() and DecodeTicketID() error mismatch")
	}

	if err1 == nil && err2 == nil {
		if decoded.EncodedID != parsed.EncodedID {
			t.Errorf("ParseTicketID() and DecodeTicketID() produced different results")
		}
	}
}

func TestParseDateFromEncoded(t *testing.T) {
	tests := []struct {
		name    string
		dateNum uint64
		wantErr bool
		want    time.Time
	}{
		{
			name:    "valid date 2025-12-25",
			dateNum: 20251225,
			wantErr: false,
			want:    time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "valid date 1970-01-01",
			dateNum: 19700101,
			wantErr: false,
			want:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "valid date 2100-12-31",
			dateNum: 21001231,
			wantErr: false,
			want:    time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:    "year too old",
			dateNum: 19690101,
			wantErr: true,
		},
		{
			name:    "year too far in future",
			dateNum: 21010101,
			wantErr: true,
		},
		{
			name:    "invalid month 0",
			dateNum: 20250001,
			wantErr: true,
		},
		{
			name:    "invalid month 13",
			dateNum: 20251301,
			wantErr: true,
		},
		{
			name:    "invalid day 0",
			dateNum: 20250100,
			wantErr: true,
		},
		{
			name:    "invalid day 32",
			dateNum: 20250132,
			wantErr: true,
		},
		{
			name:    "invalid calendar date - Feb 30",
			dateNum: 20250230,
			wantErr: true,
		},
		{
			name:    "invalid calendar date - Feb 29 non-leap year",
			dateNum: 20250229,
			wantErr: true,
		},
		{
			name:    "valid leap year date - Feb 29",
			dateNum: 20240229,
			wantErr: false,
			want:    time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDateFromEncoded(tt.dateNum)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDateFromEncoded() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if !got.Equal(tt.want) {
					t.Errorf("parseDateFromEncoded() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestFormatFunctions(t *testing.T) {
	tests := []struct {
		name     string
		num      uint64
		funcName string
		want     string
	}{
		{
			name:     "formatEventID with 0",
			num:      0,
			funcName: "formatEventID",
			want:     "EVT000000",
		},
		{
			name:     "formatEventID with 12345",
			num:      12345,
			funcName: "formatEventID",
			want:     "EVT012345",
		},
		{
			name:     "formatCategoryID with 0",
			num:      0,
			funcName: "formatCategoryID",
			want:     "CAT000000",
		},
		{
			name:     "formatCategoryID with 100",
			num:      100,
			funcName: "formatCategoryID",
			want:     "CAT000100",
		},
		{
			name:     "formatSeatID with 0",
			num:      0,
			funcName: "formatSeatID",
			want:     "SEAT000000",
		},
		{
			name:     "formatSeatID with 999",
			num:      999,
			funcName: "formatSeatID",
			want:     "SEAT000999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			switch tt.funcName {
			case "formatEventID":
				got = formatEventID(tt.num)
			case "formatCategoryID":
				got = formatCategoryID(tt.num)
			case "formatSeatID":
				got = formatSeatID(tt.num)
			}
			if got != tt.want {
				t.Errorf("%s() = %v, want %v", tt.funcName, got, tt.want)
			}
		})
	}
}

func TestIsValidTicketIDFormat(t *testing.T) {
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validTicket, _ := Generate("TEST", validDate, "CAT", "SEAT", 100)

	tests := []struct {
		name     string
		ticketID string
		want     bool
	}{
		{
			name:     "valid ticket ID",
			ticketID: validTicket,
			want:     true,
		},
		{
			name:     "valid ticket ID with dashes",
			ticketID: Format(validTicket),
			want:     true,
		},
		{
			name:     "wrong length",
			ticketID: "SHORT",
			want:     false,
		},
		{
			name:     "empty",
			ticketID: "",
			want:     false,
		},
		{
			name:     "invalid checksum",
			ticketID: validTicket[:22] + "ZZ",
			want:     false,
		},
		{
			name:     "invalid characters",
			ticketID: "ILOVEU123456789012345",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTicketIDFormat(tt.ticketID); got != tt.want {
				t.Errorf("IsValidTicketIDFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractComponents(t *testing.T) {
	validDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	validTicket, _ := Generate("TEST", validDate, "CAT", "SEAT", 100)

	tests := []struct {
		name     string
		ticketID string
		wantNil  bool
	}{
		{
			name:     "valid ticket ID",
			ticketID: validTicket,
			wantNil:  false,
		},
		{
			name:     "valid ticket ID with dashes",
			ticketID: Format(validTicket),
			wantNil:  false,
		},
		{
			name:     "wrong length",
			ticketID: "SHORT",
			wantNil:  true,
		},
		{
			name:     "empty",
			ticketID: "",
			wantNil:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractComponents(tt.ticketID)
			if (got == nil) != tt.wantNil {
				t.Errorf("ExtractComponents() nil = %v, wantNil %v", got == nil, tt.wantNil)
				return
			}
			if !tt.wantNil {
				// Verify all expected keys exist
				expectedKeys := []string{"eventID", "eventDate", "category", "seatID", "sequence", "checksum"}
				for _, key := range expectedKeys {
					if _, exists := got[key]; !exists {
						t.Errorf("ExtractComponents() missing key: %s", key)
					}
				}
				// Verify component lengths
				if len(got["eventID"]) != EventIDLength {
					t.Errorf("ExtractComponents() eventID length = %d, want %d", len(got["eventID"]), EventIDLength)
				}
				if len(got["eventDate"]) != EventDateLength {
					t.Errorf("ExtractComponents() eventDate length = %d, want %d", len(got["eventDate"]), EventDateLength)
				}
				if len(got["category"]) != CategoryLength {
					t.Errorf("ExtractComponents() category length = %d, want %d", len(got["category"]), CategoryLength)
				}
				if len(got["seatID"]) != SeatIDLength {
					t.Errorf("ExtractComponents() seatID length = %d, want %d", len(got["seatID"]), SeatIDLength)
				}
				if len(got["sequence"]) != SequenceLength {
					t.Errorf("ExtractComponents() sequence length = %d, want %d", len(got["sequence"]), SequenceLength)
				}
				if len(got["checksum"]) != ChecksumLength {
					t.Errorf("ExtractComponents() checksum length = %d, want %d", len(got["checksum"]), ChecksumLength)
				}
			}
		})
	}
}

func TestRemoveDashes(t *testing.T) {
	tests := []struct {
		name     string
		ticketID string
		want     string
	}{
		{
			name:     "with dashes",
			ticketID: "01B2M-4K6G8-N3V-9F2HA-7XJ5-QR",
			want:     "01B2M4K6G8N3V9F2HA7XJ5QR",
		},
		{
			name:     "without dashes",
			ticketID: "01B2M4K6G8N3V9F2HA7XJ5QR",
			want:     "01B2M4K6G8N3V9F2HA7XJ5QR",
		},
		{
			name:     "empty",
			ticketID: "",
			want:     "",
		},
		{
			name:     "only dashes",
			ticketID: "-----",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveDashes(tt.ticketID); got != tt.want {
				t.Errorf("RemoveDashes() = %v, want %v", got, tt.want)
			}
		})
	}
}
