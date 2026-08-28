package billingadmin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBilling_parseMoneyMicro(t *testing.T) {
	t.Parallel()
	micro := int64(1_500_000)
	v, err := ParseMoneyMicro(&micro, 0, false, "amount")
	require.NoError(t, err)
	require.Equal(t, int64(1_500_000), v)

	legacy := 2.5
	v, err = ParseMoneyMicro(nil, legacy, true, "amount")
	require.NoError(t, err)
	require.Equal(t, int64(2_500_000), v)

	neg := int64(-1)
	_, err = ParseMoneyMicro(&neg, 0, false, "amount")
	require.Error(t, err)

	_, err = ParseMoneyMicro(nil, -1, true, "amount")
	require.Error(t, err)

	v, err = ParseMoneyMicro(nil, 0, false, "amount")
	require.NoError(t, err)
	require.Equal(t, int64(0), v)
}

func TestBilling_parseBudgetMicro(t *testing.T) {
	t.Parallel()
	micro := int64(1_000_000)
	v, err := ParseBudgetMicro(&micro, 0, false)
	require.NoError(t, err)
	require.Equal(t, int64(1_000_000), v)

	v, err = ParseBudgetMicro(nil, 1.5, true)
	require.NoError(t, err)
	require.Equal(t, int64(1_500_000), v)

	_, err = ParseBudgetMicro(nil, 0, false)
	require.Error(t, err)

	zero := int64(0)
	_, err = ParseBudgetMicro(&zero, 0, false)
	require.Error(t, err)

	_, err = ParseBudgetMicro(nil, -1, true)
	require.Error(t, err)

	_, err = ParseBudgetMicro(nil, 0, true)
	require.Error(t, err)
}

func TestBilling_optionalBudgetMicro(t *testing.T) {
	t.Parallel()
	micro := int64(2_000_000)
	out, err := OptionalBudgetMicro(&micro, nil)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, int64(2_000_000), *out)

	legacy := 3.0
	out, err = OptionalBudgetMicro(nil, &legacy)
	require.NoError(t, err)
	require.NotNil(t, out)
	require.Equal(t, int64(3_000_000), *out)

	out, err = OptionalBudgetMicro(nil, nil)
	require.NoError(t, err)
	require.Nil(t, out)

	bad := int64(0)
	_, err = OptionalBudgetMicro(&bad, nil)
	require.Error(t, err)

	badLegacy := 0.0
	_, err = OptionalBudgetMicro(nil, &badLegacy)
	require.Error(t, err)
}
