package ticketid

import "fmt"

// Error represents a ticket ID operation error
type Error struct {
	Code    string
	Message string
	Details map[string]interface{}
}

// Error implements the error interface
func (e *Error) Error() string {
	if len(e.Details) > 0 {
		return fmt.Sprintf("%s: %s (details: %v)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Error codes
const (
	ErrInvalidEventID    = "INVALID_EVENT_ID"
	ErrInvalidDate       = "INVALID_DATE"
	ErrInvalidCategory   = "INVALID_CATEGORY"
	ErrInvalidSeat       = "INVALID_SEAT"
	ErrInvalidSequence   = "INVALID_SEQUENCE"
	ErrInvalidTicketID   = "INVALID_TICKET_ID"
	ErrDecodeFailed      = "DECODE_FAILED"
	ErrInvalidChecksum   = "INVALID_CHECKSUM"
	ErrInvalidLength     = "INVALID_LENGTH"
	ErrValueOutOfRange   = "VALUE_OUT_OF_RANGE"
)

// NewError creates a new Error
func NewError(code, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// NewErrorWithDetails creates a new Error with details
func NewErrorWithDetails(code, message string, details map[string]interface{}) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// WrapError wraps an existing error with a Error
func WrapError(code, message string, err error) error {
	if err == nil {
		return NewError(code, message)
	}
	return fmt.Errorf("%s: %s: %w", code, message, err)
}
