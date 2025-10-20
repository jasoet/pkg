package ticketid

import (
	"testing"
	"time"
)

func TestGenerate(t *testing.T) {
	tests := []struct {
		name       string
		eventID    string
		eventDate  time.Time
		categoryID string
		seatID     string
		sequence   int
		wantErr    bool
	}{
		{
			name:       "valid numeric IDs",
			eventID:    "12345",
			eventDate:  time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
			categoryID: "100",
			seatID:     "500",
			sequence:   12345,
			wantErr:    false,
		},
		{
			name:       "valid string IDs",
			eventID:    "EVT001",
			eventDate:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
			categoryID: "VIP",
			seatID:     "A123",
			sequence:   999,
			wantErr:    false,
		},
		{
			name:       "mixed numeric and string IDs",
			eventID:    "42",
			eventDate:  time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
			categoryID: "GENERAL",
			seatID:     "B456",
			sequence:   0,
			wantErr:    false,
		},
		{
			name:       "empty event ID",
			eventID:    "",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "empty category ID",
			eventID:    "EVT",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "empty seat ID",
			eventID:    "EVT",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "CAT",
			seatID:     "",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "negative sequence",
			eventID:    "EVT",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   -1,
			wantErr:    true,
		},
		{
			name:       "large numeric event ID (within limit)",
			eventID:    "33554431",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false,
		},
		{
			name:       "large numeric event ID (over limit)",
			eventID:    "33554432",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "large numeric category ID (within limit)",
			eventID:    "EVT",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "32767",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false,
		},
		{
			name:       "large numeric category ID (over limit)",
			eventID:    "EVT",
			eventDate:  time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			categoryID: "32768",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Generate(tt.eventID, tt.eventDate, tt.categoryID, tt.seatID, tt.sequence)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != TotalLength {
					t.Errorf("Generate() returned ticket ID length = %d, want %d", len(got), TotalLength)
				}
				// Verify format is valid
				if !IsValidTicketIDFormat(got) {
					t.Errorf("Generate() returned invalid ticket ID format: %s", got)
				}
			}
		})
	}
}

func TestGenerateSequence(t *testing.T) {
	// Test that GenerateSequence produces valid values
	for i := 0; i < 100; i++ {
		seq := GenerateSequence()
		if seq < 0 {
			t.Errorf("GenerateSequence() returned negative value: %d", seq)
		}
		if seq >= 1048576 {
			t.Errorf("GenerateSequence() returned value >= 1048576: %d", seq)
		}
	}

	// Test uniqueness (should produce different values most of the time)
	sequences := make(map[int]bool)
	for i := 0; i < 100; i++ {
		sequences[GenerateSequence()] = true
	}
	if len(sequences) < 50 { // At least 50% unique
		t.Errorf("GenerateSequence() produced too many duplicates: got %d unique out of 100", len(sequences))
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name     string
		ticketID string
		want     string
	}{
		{
			name:     "valid 24-char ticket ID",
			ticketID: "01B2M4K6G8N3V9F2HA7XJ5QR",
			want:     "01B2M-4K6G8-N3V-9F2HA-7XJ5-QR",
		},
		{
			name:     "wrong length ticket ID",
			ticketID: "SHORT",
			want:     "SHORT",
		},
		{
			name:     "empty ticket ID",
			ticketID: "",
			want:     "",
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

func TestEncodeEventID(t *testing.T) {
	tests := []struct {
		name    string
		eventID string
		wantErr bool
	}{
		{
			name:    "numeric event ID",
			eventID: "12345",
			wantErr: false,
		},
		{
			name:    "string event ID",
			eventID: "EVT001",
			wantErr: false,
		},
		{
			name:    "max valid numeric event ID",
			eventID: "33554431",
			wantErr: false,
		},
		{
			name:    "over limit numeric event ID",
			eventID: "33554432",
			wantErr: true,
		},
		{
			name:    "zero event ID",
			eventID: "0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeEventID(tt.eventID)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeEventID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != EventIDLength {
				t.Errorf("encodeEventID() returned length = %d, want %d", len(got), EventIDLength)
			}
		})
	}
}

func TestEncodeEventDate(t *testing.T) {
	tests := []struct {
		name string
		date time.Time
		want int // length
	}{
		{
			name: "valid date 2025-12-25",
			date: time.Date(2025, 12, 25, 0, 0, 0, 0, time.UTC),
			want: EventDateLength,
		},
		{
			name: "valid date 1970-01-01",
			date: time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
			want: EventDateLength,
		},
		{
			name: "valid date 2100-12-31",
			date: time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC),
			want: EventDateLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeEventDate(tt.date)
			if len(got) != tt.want {
				t.Errorf("encodeEventDate() length = %d, want %d", len(got), tt.want)
			}
		})
	}
}

func TestEncodeCategoryID(t *testing.T) {
	tests := []struct {
		name       string
		categoryID string
		wantErr    bool
	}{
		{
			name:       "numeric category ID",
			categoryID: "100",
			wantErr:    false,
		},
		{
			name:       "string category ID",
			categoryID: "VIP",
			wantErr:    false,
		},
		{
			name:       "max valid numeric category ID",
			categoryID: "32767",
			wantErr:    false,
		},
		{
			name:       "over limit numeric category ID",
			categoryID: "32768",
			wantErr:    true,
		},
		{
			name:       "zero category ID",
			categoryID: "0",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeCategoryID(tt.categoryID)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeCategoryID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != CategoryLength {
				t.Errorf("encodeCategoryID() returned length = %d, want %d", len(got), CategoryLength)
			}
		})
	}
}

func TestEncodeSeatID(t *testing.T) {
	tests := []struct {
		name    string
		seatID  string
		wantErr bool
	}{
		{
			name:    "numeric seat ID",
			seatID:  "500",
			wantErr: false,
		},
		{
			name:    "string seat ID",
			seatID:  "A123",
			wantErr: false,
		},
		{
			name:    "max valid numeric seat ID",
			seatID:  "33554431",
			wantErr: false,
		},
		{
			name:    "over limit numeric seat ID",
			seatID:  "33554432",
			wantErr: true,
		},
		{
			name:    "zero seat ID",
			seatID:  "0",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := encodeSeatID(tt.seatID)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeSeatID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) != SeatIDLength {
				t.Errorf("encodeSeatID() returned length = %d, want %d", len(got), SeatIDLength)
			}
		})
	}
}

func TestEncodeSequence(t *testing.T) {
	tests := []struct {
		name     string
		sequence int
		wantLen  int
	}{
		{
			name:     "zero sequence",
			sequence: 0,
			wantLen:  SequenceLength,
		},
		{
			name:     "positive sequence",
			sequence: 12345,
			wantLen:  SequenceLength,
		},
		{
			name:     "max sequence",
			sequence: 1048575,
			wantLen:  SequenceLength,
		},
		{
			name:     "over max sequence (should wrap)",
			sequence: 1048576,
			wantLen:  SequenceLength,
		},
		{
			name:     "negative sequence (should become 0)",
			sequence: -100,
			wantLen:  SequenceLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeSequence(tt.sequence)
			if len(got) != tt.wantLen {
				t.Errorf("encodeSequence() length = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestHashStringToUint64(t *testing.T) {
	tests := []struct {
		name string
		s    string
	}{
		{
			name: "simple string",
			s:    "test",
		},
		{
			name: "empty string",
			s:    "",
		},
		{
			name: "long string",
			s:    "this is a very long string with many characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := hashStringToUint64(tt.s)
			// Just verify it returns a value (hash function should always work)
			_ = got
		})
	}

	// Test consistency - same input should produce same hash
	s := "consistent"
	hash1 := hashStringToUint64(s)
	hash2 := hashStringToUint64(s)
	if hash1 != hash2 {
		t.Errorf("hashStringToUint64() not consistent: %d != %d", hash1, hash2)
	}

	// Test different inputs produce different hashes
	hash3 := hashStringToUint64("different")
	if hash1 == hash3 {
		t.Errorf("hashStringToUint64() collision: same hash for different strings")
	}
}

func TestValidateGenerateInputs(t *testing.T) {
	tests := []struct {
		name       string
		eventID    string
		categoryID string
		seatID     string
		sequence   int
		wantErr    bool
	}{
		{
			name:       "all valid",
			eventID:    "EVT",
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    false,
		},
		{
			name:       "empty event ID",
			eventID:    "",
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "empty category ID",
			eventID:    "EVT",
			categoryID: "",
			seatID:     "SEAT",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "empty seat ID",
			eventID:    "EVT",
			categoryID: "CAT",
			seatID:     "",
			sequence:   1,
			wantErr:    true,
		},
		{
			name:       "negative sequence",
			eventID:    "EVT",
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   -1,
			wantErr:    true,
		},
		{
			name:       "zero sequence",
			eventID:    "EVT",
			categoryID: "CAT",
			seatID:     "SEAT",
			sequence:   0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateGenerateInputs(tt.eventID, tt.categoryID, tt.seatID, tt.sequence)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateGenerateInputs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
