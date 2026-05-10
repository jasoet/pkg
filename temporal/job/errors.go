package job

import (
	"errors"
	"fmt"

	"go.temporal.io/api/serviceerror"
)

var (
	// Lookup
	ErrNotFound          = errors.New("job: not found")
	ErrDuplicateName     = errors.New("job: duplicate name")
	ErrInvalidDefinition = errors.New("job: invalid definition")

	// Lifecycle
	ErrAlreadyClosed    = errors.New("job: workflow already closed")
	ErrNoSchedule       = errors.New("job: no schedule configured")
	ErrScheduleNotFound = errors.New("job: schedule not found")

	// Wiring
	ErrNotRegistered = errors.New("job: register not configured")
)

// translateSDKError wraps a Temporal SDK error with a typed sentinel where a
// matching one exists; otherwise wraps with the operation name for context.
// Always preserves the original error chain so callers can errors.As to the
// SDK types when needed.
func translateSDKError(op string, err error) error {
	if err == nil {
		return nil
	}
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return fmt.Errorf("%s: %w: %w", op, ErrNotFound, err)
	}
	return fmt.Errorf("%s: %w", op, err)
}
