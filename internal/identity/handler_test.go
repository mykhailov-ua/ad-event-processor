package identity

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandler_RegisterRequiresAdminKey(t *testing.T) {
	handler := NewHandler(nil, &config.Config{AdminAPIKey: "secret-admin-key"})

	_, err := handler.API().Register(context.Background(), "", "user@example.com", "Password123!", "U", "")
	require.Error(t, err)
	assert.Equal(t, ErrInvalidCredentials, err)
}
