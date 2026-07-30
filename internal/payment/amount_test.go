package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripeAmountToMicro(t *testing.T) {
	assert.Equal(t, int64(50_000_000), StripeAmountToMicro(5000))
	assert.Equal(t, int64(10_000_000), StripeAmountToMicro(1000))
	assert.Equal(t, int64(10_000), StripeAmountToMicro(1))
}

func TestMicroToStripeAmount(t *testing.T) {
	cents, err := MicroToStripeAmount(50_000_000)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), cents)

	_, err = MicroToStripeAmount(10_001)
	assert.Error(t, err)
}
