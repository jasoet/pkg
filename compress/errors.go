package compress

import "errors"

var (
	ErrSizeLimitExceeded = errors.New("size limit exceeded")
	ErrPathTraversal     = errors.New("path traversal detected")
)
