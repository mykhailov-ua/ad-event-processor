package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBilling_parseMoneyMicro(t *testing.T) {
	t.Parallel()
	micro := int64(1_500_000)
	v, err := parseMoneyMicro(&micro, 0, false, "amount")
	require.NoError(t, err)
	assert.Equal(t, int64(1_500_000), v)

	legacy := 2.5
	v, err = parseMoneyMicro(nil, legacy, true, "amount")
	require.NoError(t, err)
	assert.Equal(t, int64(2_500_000), v)

	neg := int64(-1)
	_, err = parseMoneyMicro(&neg, 0, false, "amount")
	require.Error(t, err)
}

func TestBilling_parseBudgetMicro(t *testing.T) {
	t.Parallel()
	micro := int64(5_000_000)
	v, err := parseBudgetMicro(&micro, 0, false)
	require.NoError(t, err)
	assert.Equal(t, int64(5_000_000), v)

	_, err = parseBudgetMicro(nil, 0, false)
	require.Error(t, err)

	zero := int64(0)
	_, err = parseBudgetMicro(&zero, 0, false)
	require.Error(t, err)
}

func TestBilling_optionalBudgetMicro(t *testing.T) {
	t.Parallel()
	micro := int64(1_000_000)
	v, err := optionalBudgetMicro(&micro, nil)
	require.NoError(t, err)
	require.NotNil(t, v)
	assert.Equal(t, int64(1_000_000), *v)

	legacy := 1.5
	v, err = optionalBudgetMicro(nil, &legacy)
	require.NoError(t, err)
	require.NotNil(t, v)

	out, err := optionalBudgetMicro(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	bad := int64(-1)
	_, err = optionalBudgetMicro(&bad, nil)
	require.Error(t, err)
}

func TestBilling_parseMoneyMicro_legacyInvalid(t *testing.T) {
	t.Parallel()
	_, err := parseMoneyMicro(nil, -1, true, "amount")
	require.Error(t, err)
	_, err = parseMoneyMicro(nil, 0, false, "amount")
	require.NoError(t, err)
	assert.Equal(t, int64(0), mustMicro(t, nil, 0, false))
}

func mustMicro(t *testing.T, micro *int64, legacy float64, hasLegacy bool) int64 {
	t.Helper()
	v, err := parseMoneyMicro(micro, legacy, hasLegacy, "amount")
	require.NoError(t, err)
	return v
}

func TestBilling_parseBudgetMicro_legacy(t *testing.T) {
	t.Parallel()
	v, err := parseBudgetMicro(nil, 10.0, true)
	require.NoError(t, err)
	assert.Equal(t, int64(10_000_000), v)
}

func TestBilling_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "billing", FileDomain("billing_money.go"))
}
