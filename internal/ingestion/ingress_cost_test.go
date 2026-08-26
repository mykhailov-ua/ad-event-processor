package ingestion

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
	parsed := &clickQueryParsed{ingressCost: []byte("0.05")}
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
	parsed := &clickQueryParsed{ingressCost: []byte("0.05")}
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
	parsed := &clickQueryParsed{ingressCPC: []byte("125000")}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(125_000), evt.IngressCostMicro)
}

func TestAttachIngressCost_disabledConfig(t *testing.T) {
	camp := &domain.Campaign{}
	evt := &domain.Event{}
	parsed := &clickQueryParsed{ingressCost: []byte("0.05")}
	attachIngressCost(evt, camp, parsed)
	require.Equal(t, int64(0), evt.IngressCostMicro)
}
