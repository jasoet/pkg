package rest_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jasoet/pkg/v3/rest"
)

func TestResponse_Predicates(t *testing.T) {
	cases := []struct {
		code                                               int
		serverErr, authErr, notFound, clientErr, isErr, ok bool
	}{
		{200, false, false, false, false, false, true},
		{401, false, true, false, true, true, false},
		{403, false, true, false, true, true, false},
		{404, false, false, true, true, true, false},
		{500, true, false, false, false, true, false},
	}
	for _, tc := range cases {
		r := &rest.Response{StatusCode: tc.code, Header: http.Header{}}
		assert.Equal(t, tc.serverErr, r.IsServerError(), "code %d", tc.code)
		assert.Equal(t, tc.authErr, r.IsAuthError(), "code %d", tc.code)
		assert.Equal(t, tc.notFound, r.IsNotFound(), "code %d", tc.code)
		assert.Equal(t, tc.clientErr, r.IsClientError(), "code %d", tc.code)
		assert.Equal(t, tc.isErr, r.IsError(), "code %d", tc.code)
		assert.Equal(t, tc.ok, r.IsSuccess(), "code %d", tc.code)
	}
}
