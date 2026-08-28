package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/outbox"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleLanderPublished_publishFlowReloadNilRedis(t *testing.T) {
	t.Parallel()
	payload := []byte(`{"lander_id":"` + uuid.New().String() + `"}`)
	require.NoError(t, outbox.PublishFlowReload(context.Background(), nil, "flow:reload-test"))
	_ = payload
}

func TestLanderPublicBase_envOverride(t *testing.T) {
	t.Parallel()
	svc := &Service{cfg: &config.Config{LanderPublicBaseURL: "https://edge.example"}}
	assert.Equal(t, "https://edge.example", svc.landerPublicBase(context.Background()))
}
