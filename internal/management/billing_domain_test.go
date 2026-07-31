package management

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBilling_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "billing", FileDomain("billing_money.go"))
}

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

	_, err = parseMoneyMicro(nil, -1, true, "amount")
	require.Error(t, err)

	v, err = parseMoneyMicro(nil, 0, false, "amount")
	require.NoError(t, err)
	assert.Equal(t, int64(0), v)
}

func TestBilling_parseBudgetMicro(t *testing.T) {
	t.Parallel()
	micro := int64(1_000_000)
	v, err := parseBudgetMicro(&micro, 0, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1_000_000), v)

	v, err = parseBudgetMicro(nil, 1.5, true)
	require.NoError(t, err)
	assert.Equal(t, int64(1_500_000), v)

	_, err = parseBudgetMicro(nil, 0, false)
	require.Error(t, err)

	zero := int64(0)
	_, err = parseBudgetMicro(&zero, 0, false)
	require.Error(t, err)

	_, err = parseBudgetMicro(nil, -1, true)
	require.Error(t, err)

	_, err = parseBudgetMicro(nil, 0, true)
	require.Error(t, err)
}

func TestBilling_optionalBudgetMicro(t *testing.T) {
	t.Parallel()
	micro := int64(2_000_000)
	out, err := optionalBudgetMicro(&micro, nil)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, int64(2_000_000), *out)

	legacy := 3.0
	out, err = optionalBudgetMicro(nil, &legacy)
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, int64(3_000_000), *out)

	out, err = optionalBudgetMicro(nil, nil)
	require.NoError(t, err)
	assert.Nil(t, out)

	bad := int64(0)
	_, err = optionalBudgetMicro(&bad, nil)
	require.Error(t, err)

	badLegacy := -1.0
	_, err = optionalBudgetMicro(nil, &badLegacy)
	require.Error(t, err)
}
