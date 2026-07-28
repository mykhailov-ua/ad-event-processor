package dedupkey

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeSpendSyncPayload_RoundTrip(t *testing.T) {
	camp := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	txns := make([]SpendSyncTxn, 100)
	for i := range txns {
		txns[i] = SpendSyncTxn{
			CampaignID:  camp,
			AmountMicro: int64(i + 1),
			TxnID:       fmt.Sprintf("txn-%d", i),
		}
	}
	raw, err := EncodeSpendSyncPayload(txns)
	require.NoError(t, err)
	got, err := DecodeSpendSyncPayload(raw)
	require.NoError(t, err)
	require.Equal(t, txns, got)
}

func TestDecodeSpendSyncPayload_RejectsWrongKind(t *testing.T) {
	_, err := DecodeSpendSyncPayload([]byte(`{"kind":"other","txns":[]}`))
	require.Error(t, err)
}
