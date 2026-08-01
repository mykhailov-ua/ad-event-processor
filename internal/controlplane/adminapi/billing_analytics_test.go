package adminapi

import (
	"espx/internal/ledger"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestPlaceholderProvider_walletFields(t *testing.T) {
	t.Parallel()
	p := ledger.NewPaymentProvider("placeholder", "placeholder_dev")
	require.NotNil(t, p)
	assert.Equal(t, "placeholder", p.Name())
	assert.False(t, p.Configured())
}

func TestParseStatementPeriod_month(t *testing.T) {
	t.Parallel()
	from, to, err := ParseStatementPeriod("", "", "2026-06")
	require.NoError(t, err)
	assert.Equal(t, 2026, from.Year())
	assert.Equal(t, time.June, from.Month())
	assert.Equal(t, time.July, to.Month())
}
