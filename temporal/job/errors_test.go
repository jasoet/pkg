package job

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.temporal.io/api/serviceerror"
)

func TestTranslateSDKError_NotFound(t *testing.T) {
	sdk := serviceerror.NewNotFound("workflow not found")
	got := translateSDKError("describe", sdk)
	assert.True(t, errors.Is(got, ErrNotFound))
	assert.True(t, errors.Is(got, sdk), "preserves underlying SDK error")
}

func TestTranslateSDKError_Passthrough(t *testing.T) {
	other := errors.New("plain error")
	got := translateSDKError("cancel", other)
	assert.False(t, errors.Is(got, ErrNotFound))
	assert.True(t, errors.Is(got, other))
}

func TestTranslateSDKError_Nil(t *testing.T) {
	got := translateSDKError("op", nil)
	assert.NoError(t, got)
}
