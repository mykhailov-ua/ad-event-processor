package campaign

import (
	"context"
	"fmt"
	"math"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func OptimizeBrandCreativeMABTx(ctx context.Context, tx pgx.Tx, host DeliveryHost) ([]uuid.UUID, error) {
	if host == nil {
		return nil, nil
	}
	minImps := host.MABMinImpressions()
	lookbackDays := host.MABLookbackDays()

	q := db.New(tx)
	allCreatives, err := q.ListAllActiveBrandCreatives(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active brand creatives: %w", err)
	}
	if len(allCreatives) == 0 {
		return nil, nil
	}
	campaignRows, err := q.ListCampaignIDsForActiveBrands(ctx)
	if err != nil {
		return nil, fmt.Errorf("list campaigns for active brands: %w", err)
	}

	creativesByBrand := make(map[pgtype.UUID][]db.BrandCreative)
	for _, cr := range allCreatives {
		creativesByBrand[cr.BrandID] = append(creativesByBrand[cr.BrandID], cr)
	}
	campaignsByBrand := make(map[pgtype.UUID][]pgtype.UUID)
	for _, row := range campaignRows {
		campaignsByBrand[row.BrandID] = append(campaignsByBrand[row.BrandID], row.ID)
	}

	lookbackEnd := time.Now().UTC()
	lookbackStart := lookbackEnd.Add(-time.Duration(lookbackDays) * 24 * time.Hour)
	clickhouseStats, err := host.QueryMABCreativeStats(ctx, lookbackStart, lookbackEnd)
	if err != nil {
		return nil, err
	}
	if clickhouseStats == nil {
		return nil, nil
	}

	var updatedBrands []uuid.UUID
	weightBatch := &pgx.Batch{}
	for brandID, creatives := range creativesByBrand {
		if len(creatives) < 2 {
			continue
		}

		attributed := attributeMABStats(creatives, campaignsByBrand[brandID], clickhouseStats, minImps)
		if !attributed.anyEligible {
			continue
		}

		newWeights := computeMABWeights(attributed.perCreative)
		brandChanged := false
		for _, cr := range creatives {
			creativeID := uuid.UUID(cr.ID.Bytes)
			newWeight, ok := newWeights[creativeID]
			if !ok || newWeight == cr.Weight {
				continue
			}
			weightBatch.Queue(
				`UPDATE brand_creatives SET weight = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
				cr.ID,
				newWeight,
			)
			brandChanged = true
		}
		if brandChanged {
			updatedBrands = append(updatedBrands, uuid.UUID(brandID.Bytes))
		}
	}
	if weightBatch.Len() > 0 {
		br := tx.SendBatch(ctx, weightBatch)
		defer func() { _ = br.Close() }()
		for i := 0; i < weightBatch.Len(); i++ {
			if _, err := br.Exec(); err != nil {
				return nil, fmt.Errorf("update creative weight batch item %d: %w", i, err)
			}
		}
	}
	return updatedBrands, nil
}

type mabAttribution struct {
	perCreative map[uuid.UUID]CreativeMABStat
	anyEligible bool
}

func attributeMABStats(
	creatives []db.BrandCreative,
	campaignRows []pgtype.UUID,
	clickhouseStats map[uuid.UUID]CreativeMABStat,
	minImps int64,
) mabAttribution {
	out := mabAttribution{perCreative: make(map[uuid.UUID]CreativeMABStat, len(creatives))}

	for creativeID, stat := range clickhouseStats {
		if stat.Impressions >= minImps {
			out.perCreative[creativeID] = stat
			out.anyEligible = true
		}
	}
	if out.anyEligible {
		return out
	}

	if len(creatives) == 0 || len(campaignRows) == 0 {
		return out
	}

	var totalImps, totalClicks int64
	for _, camp := range campaignRows {
		if stat, ok := clickhouseStats[uuid.UUID(camp.Bytes)]; ok {
			totalImps += stat.Impressions
			totalClicks += stat.Clicks
		}
	}
	if totalImps < minImps {
		return out
	}

	shareImps := totalImps / int64(len(creatives))
	shareClicks := totalClicks / int64(len(creatives))
	if shareImps < minImps {
		return out
	}

	for _, cr := range creatives {
		creativeID := uuid.UUID(cr.ID.Bytes)
		out.perCreative[creativeID] = CreativeMABStat{
			Impressions: shareImps,
			Clicks:      shareClicks,
		}
	}
	out.anyEligible = true
	return out
}

func ComputeMABWeights(stats map[uuid.UUID]CreativeMABStat) map[uuid.UUID]int32 {
	return computeMABWeights(stats)
}

func computeMABWeights(stats map[uuid.UUID]CreativeMABStat) map[uuid.UUID]int32 {
	weights := make(map[uuid.UUID]int32, len(stats))
	var sumCTR float64
	for _, stat := range stats {
		if stat.Impressions > 0 {
			sumCTR += float64(stat.Clicks) / float64(stat.Impressions)
		}
	}
	if sumCTR <= 0 {
		for id := range stats {
			weights[id] = 1
		}
		return weights
	}
	for id, stat := range stats {
		ctr := float64(stat.Clicks) / float64(stat.Impressions)
		w := int32(math.Max(1, math.Round(100*ctr/sumCTR)))
		weights[id] = w
	}
	return weights
}
