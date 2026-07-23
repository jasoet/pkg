package compress

import "errors"

var (
	// ErrSizeLimitExceeded is returned when an extraction guard rail rejects
	// content that exceeds a configured size limit (per-file or total archive).
	ErrSizeLimitExceeded = errors.New("size limit exceeded")
	// ErrPathTraversal is returned when a path guard rail rejects a destination
	// or archive entry that could escape the intended extraction location.
	ErrPathTraversal = errors.New("path traversal detected")
	// ErrNotDirectory is returned when a path that must be a directory
	// (tar source or extraction destination) is not one.
	ErrNotDirectory = errors.New("not a directory")
)
