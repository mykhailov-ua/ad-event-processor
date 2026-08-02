package controlplane

import (
	"testing"

	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestOutboxProto_roundTripCampaign(t *testing.T) {
	in := CampaignPayload{CampaignID: uuid.New().String(), BudgetLimit: 1_000_000}
	raw, err := coldpath.MarshalOutbox(in)
	require.NoError(t, err)
	require.True(t, coldpath.IsOutboxProto(raw))

	out, err := coldpath.UnmarshalStrict[CampaignPayload](raw)
	require.NoError(t, err)
	require.Equal(t, in, out)
}

func TestOutboxProto_legacyJSONCampaign(t *testing.T) {
	legacy := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","budget_limit":500}`)
	out, err := coldpath.UnmarshalStrict[CampaignPayload](legacy)
	require.NoError(t, err)
	require.Equal(t, int64(500), out.BudgetLimit)
}

func BenchmarkOutboxDecode_CampaignProto(b *testing.B) {
	in := CampaignPayload{CampaignID: uuid.New().String(), BudgetLimit: 100_500_000}
	raw, err := coldpath.MarshalOutbox(in)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := coldpath.UnmarshalStrict[CampaignPayload](raw)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOutboxDecode_CampaignJSON(b *testing.B) {
	raw := []byte(`{"campaign_id":"00000000-0000-0000-0000-000000000001","budget_limit":100500000}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := coldpath.UnmarshalStrict[CampaignPayload](raw)
		if err != nil {
			b.Fatal(err)
		}
	}
}
