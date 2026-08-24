package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadBudgetInvariants_emptyIDs(t *testing.T) {
	snaps, err := ReadBudgetInvariants(context.Background(), nil, nil, nil)
	require.NoError(t, err)
	assert.Empty(t, snaps)
}
