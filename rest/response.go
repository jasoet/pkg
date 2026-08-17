package rest

import (
	"net/http"

	"github.com/go-resty/resty/v2"
)

// Response is the library-owned HTTP response returned by MakeRequest and
// MakeRequestWithTrace. It decouples callers from the underlying resty types.
type Response struct {
	StatusCode int
	Body       string
	Header     http.Header
}

// IsError returns true for any HTTP status code >= 400.
func (r *Response) IsError() bool {
	return r.StatusCode >= 400
}

// IsSuccess returns true for HTTP 2xx status codes.
func (r *Response) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// IsServerError returns true for HTTP 5xx status codes.
func (r *Response) IsServerError() bool {
	return r.StatusCode >= 500
}

// IsAuthError returns true for HTTP 401 (Unauthorized) and 403 (Forbidden).
// Both indicate an access control failure and both map to UnauthorizedError
// in handleResponse; use StatusCode to distinguish them.
func (r *Response) IsAuthError() bool {
	return r.StatusCode == http.StatusUnauthorized || r.StatusCode == http.StatusForbidden
}

// IsNotFound returns true for HTTP 404 (Not Found).
func (r *Response) IsNotFound() bool {
	return r.StatusCode == http.StatusNotFound
}

// IsClientError returns true for any HTTP 4xx status code.
// Note: this overlaps with IsAuthError and IsNotFound; in handleResponse,
// those are checked first so IsClientError only catches remaining 4xx codes.
func (r *Response) IsClientError() bool {
	return r.StatusCode >= 400 && r.StatusCode < 500
}

// fromResty converts a resty response to the library-owned Response.
// A nil resty response yields a nil Response.
func fromResty(resp *resty.Response) *Response {
	if resp == nil {
		return nil
	}
	return &Response{
		StatusCode: resp.StatusCode(),
		Body:       resp.String(),
		Header:     resp.Header(),
	}
}
