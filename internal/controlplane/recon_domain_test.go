package controlplane

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecon_parseReconciliationAdjustPayload_valid(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"campaign_id":"` + uuid.New().String() + `","customer_id":"` + uuid.New().String() + `","ledger_amount_micro":100,"redis_delta_micro":0}`)
	p, err := parseReconciliationAdjustPayload(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(100), p.LedgerAmt)
}

func TestRecon_parseReconciliationAdjustPayload_rejectsEmpty(t *testing.T) {
	t.Parallel()
	_, err := parseReconciliationAdjustPayload([]byte(`{"campaign_id":"","customer_id":"x","ledger_amount_micro":1}`))
	require.Error(t, err)

	_, err = parseReconciliationAdjustPayload([]byte(`{"campaign_id":"` + uuid.New().String() + `","customer_id":"` + uuid.New().String() + `"}`))
	require.Error(t, err)
}

func TestRecon_parseReconciliationAdjustPayload_invalidJSON(t *testing.T) {
	t.Parallel()
	_, err := parseReconciliationAdjustPayload([]byte("{"))
	require.Error(t, err)
}

func TestRecon_parseReconciliationAdjustPayload_redisOnly(t *testing.T) {
	t.Parallel()
	raw := []byte(`{"campaign_id":"` + uuid.New().String() + `","customer_id":"` + uuid.New().String() + `","redis_delta_micro":100}`)
	p, err := parseReconciliationAdjustPayload(raw)
	require.NoError(t, err)
	assert.Equal(t, int64(100), p.RedisDelta)
}
