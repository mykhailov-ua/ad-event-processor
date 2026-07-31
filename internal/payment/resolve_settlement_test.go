package payment

import (
	"context"
	"testing"

	"espx/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSettlementAPIOrDial_GRPCDisabled(t *testing.T) {
	cfg := &config.Config{SettlementGRPCEnabled: false}
	api, closeFn, err := OpenSettlementAPIOrDial(context.Background(), cfg)
	require.NoError(t, err)
	assert.Nil(t, api)
	require.NotNil(t, closeFn)
	closeFn()
}
