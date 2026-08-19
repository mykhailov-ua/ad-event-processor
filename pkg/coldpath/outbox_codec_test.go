package coldpath

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type benchCampaignPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}

func TestMarshalOutbox_jsonFallback(t *testing.T) {
	in := benchCampaignPayload{CampaignID: "00000000-0000-0000-0000-000000000001", BudgetLimit: 42}
	raw, err := MarshalOutbox(in)
	require.NoError(t, err)
	require.False(t, IsOutboxProto(raw))

	var out benchCampaignPayload
	require.NoError(t, UnmarshalJSON(raw, &out))
	require.Equal(t, in, out)
}

func TestOutboxProtoRoundTrip_registeredType(t *testing.T) {
	RegisterOutboxCodec(
		func(p benchCampaignPayload) ([]byte, error) {
			return MarshalJSON(p)
		},
		func(b []byte) (benchCampaignPayload, error) {
			var p benchCampaignPayload
			err := UnmarshalJSON(b, &p)
			return p, err
		},
	)

	in := benchCampaignPayload{CampaignID: "abc", BudgetLimit: 99}
	raw, err := MarshalOutbox(in)
	require.NoError(t, err)
	require.True(t, IsOutboxProto(raw))

	out, err := UnmarshalStrict[benchCampaignPayload](raw)
	require.NoError(t, err)
	require.Equal(t, in, out)
}

func TestUnmarshalStrict_legacyJSON(t *testing.T) {
	raw := []byte(`{"campaign_id":"x","budget_limit":1}`)
	RegisterOutboxCodec(
		func(p benchCampaignPayload) ([]byte, error) { return MarshalJSON(p) },
		func(b []byte) (benchCampaignPayload, error) {
			var p benchCampaignPayload
			err := UnmarshalJSON(b, &p)
			return p, err
		},
	)
	out, err := UnmarshalStrict[benchCampaignPayload](raw)
	require.NoError(t, err)
	require.Equal(t, "x", out.CampaignID)
}
