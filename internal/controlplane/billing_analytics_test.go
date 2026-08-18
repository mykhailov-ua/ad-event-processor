package controlplane

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseStatementPeriod_month(t *testing.T) {
	t.Parallel()
	from, to, err := ParseStatementPeriod("", "", "2026-06")
	require.NoError(t, err)
	assert.Equal(t, 2026, from.Year())
	assert.Equal(t, time.June, from.Month())
	assert.Equal(t, time.July, to.Month())
}
