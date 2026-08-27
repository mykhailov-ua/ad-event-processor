package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTelegramPostbackRelayContext_respectsParentCancel(t *testing.T) {
	t.Parallel()
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	relayCtx, relayCancel := telegramPostbackRelayContext(parent)
	defer relayCancel()
	require.Error(t, relayCtx.Err())
}

func TestTelegramPostbackRelayContext_timeout(t *testing.T) {
	t.Parallel()
	require.Equal(t, 30*time.Second, telegramPostbackRelayTimeout)
}

func TestTelegramService_relayPostbacks_respectsCanceledContext(t *testing.T) {
	t.Parallel()
	svc := NewTelegramService(&Service{}, nil, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	svc.relayPostbacks(ctx, uuid.New(), "click-token")
}
