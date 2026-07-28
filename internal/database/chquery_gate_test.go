package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCHQuery_acquireRejectWhenSaturated(t *testing.T) {
	t.Parallel()

	q := &CHQuery{sem: make(chan struct{}, 1)}
	q.sem <- struct{}{}

	err := q.acquire(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCHQueryRejected)
}

func TestCHQueryConfigFromApp_defaults(t *testing.T) {
	t.Parallel()

	cfg := CHQueryConfigFromApp(nil)
	assert.Equal(t, CHQueryConfig{}, cfg)
}
