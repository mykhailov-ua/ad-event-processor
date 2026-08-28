package campaign

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/supply"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SupplyChainHost interface {
	MapCampaignNotFound(err error) error
	AuditSupplyChainUpdate(ctx context.Context, q db.Querier, campaignID uuid.UUID, oldNodesJSON, newNodesJSON []byte)
}

func GetCampaignSupplyChain(ctx context.Context, pool *pgxpool.Pool, host SupplyChainHost, campaignID uuid.UUID) (supply.CampaignChainDTO, error) {
	if pool == nil || host == nil {
		return supply.CampaignChainDTO{}, errServiceUnavailable()
	}
	row, err := db.New(pool).GetCampaignFull(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return supply.CampaignChainDTO{}, host.MapCampaignNotFound(err)
	}
	nodes, err := parseSupplyChainNodes(row.SupplyChainNodes)
	if err != nil {
		return supply.CampaignChainDTO{}, err
	}
	return supply.CampaignChainDTO{CampaignID: campaignID.String(), Nodes: nodes}, nil
}

func UpdateCampaignSupplyChain(ctx context.Context, pool *pgxpool.Pool, host SupplyChainHost, campaignID uuid.UUID, nodes []supply.ChainNode) (supply.CampaignChainDTO, error) {
	if err := supply.ValidateChainNodes(nodes); err != nil {
		return supply.CampaignChainDTO{}, err
	}
	if pool == nil || host == nil {
		return supply.CampaignChainDTO{}, errServiceUnavailable()
	}
	nodesJSON, err := coldpath.MarshalJSON(nodes)
	if err != nil {
		return supply.CampaignChainDTO{}, err
	}
	var out supply.CampaignChainDTO
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		locked, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return err
		}
		oldNodes, _ := parseSupplyChainNodes(locked.SupplyChainNodes)
		oldNodesJSON, err := coldpath.MarshalJSON(oldNodes)
		if err != nil {
			return err
		}
		updated, err := q.UpdateCampaignSupplyChain(ctx, db.UpdateCampaignSupplyChainParams{
			ID:               domain.ToUUID(campaignID),
			SupplyChainNodes: nodesJSON,
		})
		if err != nil {
			return err
		}
		host.AuditSupplyChainUpdate(ctx, q, campaignID, oldNodesJSON, nodesJSON)
		parsed, err := parseSupplyChainNodes(updated.SupplyChainNodes)
		if err != nil {
			return err
		}
		out = supply.CampaignChainDTO{CampaignID: campaignID.String(), Nodes: parsed}
		return nil
	})
	return out, err
}

func parseSupplyChainNodes(raw []byte) ([]supply.ChainNode, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return []supply.ChainNode{}, nil
	}
	var nodes []supply.ChainNode
	if err := coldpath.UnmarshalJSON(raw, &nodes); err != nil {
		return nil, err
	}
	return nodes, nil
}
