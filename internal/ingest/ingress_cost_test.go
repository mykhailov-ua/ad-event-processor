package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAttachIngressCost_decimalCostParam(t *testing.T) {
	camp := &domain.Campaign{
		ID: uuid.New(),
		IngressCost: domain.IngressCostConfig{
			Param:    domain.IngressCostParamCost,
			MaxMicro: 10_000_000,
		},
	}
	evt := &domain.Event{CampaignID: camp.ID}
	parsed := &clickQueryParsed{IngressCost: []byte("0.05")}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(50_000), evt.IngressCostMicro)
}

func TestAttachIngressCost_overCapIgnored(t *testing.T) {
	camp := &domain.Campaign{
		IngressCost: domain.IngressCostConfig{
			Param:    domain.IngressCostParamCost,
			MaxMicro: 10_000,
		},
	}
	evt := &domain.Event{}
	parsed := &clickQueryParsed{IngressCost: []byte("0.05")}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(0), evt.IngressCostMicro)
}

func TestAttachIngressCost_microScale(t *testing.T) {
	camp := &domain.Campaign{
		IngressCost: domain.IngressCostConfig{
			Param:      domain.IngressCostParamCPC,
			ScaleMicro: true,
		},
	}
	evt := &domain.Event{}
	parsed := &clickQueryParsed{IngressCPC: []byte("125000")}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(125_000), evt.IngressCostMicro)
}

func TestAttachIngressCost_disabledConfig(t *testing.T) {
	camp := &domain.Campaign{}
	evt := &domain.Event{}
	parsed := &clickQueryParsed{IngressCost: []byte("0.05")}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(0), evt.IngressCostMicro)
}

func TestAttachIngressCost_patchJSONShape_holdout(t *testing.T) {
	camp := &domain.Campaign{
		ID: uuid.New(),
		IngressCost: domain.ParseIngressCostConfigJSON(
			[]byte(`{"param":"cost","scale":"decimal","max_micro":5000000,"policy":"ignore"}`),
		),
	}
	parsed := &clickQueryParsed{}
	path := []byte("/click?campaign_id=" + camp.ID.String() + "&type=click&cost=0.12")
	parseClickQuery(path, nil, parsed)
	require.True(t, parsed.OK)

	evt := &domain.Event{CampaignID: camp.ID}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(120_000), evt.IngressCostMicro)
	require.Equal(t, ingressCostSourceMacro, clickAttributedCostSource(evt))
}

func TestAttachIngressCost_withoutPatchConfig_holdout(t *testing.T) {
	camp := &domain.Campaign{ID: uuid.New()}
	parsed := &clickQueryParsed{}
	path := []byte("/click?campaign_id=" + camp.ID.String() + "&type=click&cost=0.12")
	parseClickQuery(path, nil, parsed)
	require.True(t, parsed.OK)

	evt := &domain.Event{CampaignID: camp.ID}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(0), evt.IngressCostMicro)
	require.Empty(t, clickAttributedCostSource(evt))
}
