package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrPublisherScopeRequired = errors.New("publisher seller_id bind required")

func (s *Service) ResolvePublisherBind(ctx context.Context, userID uuid.UUID) (PublisherBind, error) {
	pool := s.GetPool()
	if pool == nil {
		return PublisherBind{}, errors.New("publisher service unavailable")
	}
	var sellerID, pubAcct *string
	var customerID uuid.UUID
	err := pool.QueryRow(ctx, `
		SELECT seller_id, publisher_account_id, customer_id
		FROM users
		WHERE id = $1`, userID).Scan(&sellerID, &pubAcct, &customerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return PublisherBind{}, ErrPublisherScopeRequired
		}
		return PublisherBind{}, err
	}
	bind := PublisherBind{CustomerID: customerID}
	if sellerID != nil {
		bind.SellerID = strings.TrimSpace(*sellerID)
	}
	if pubAcct != nil {
		bind.PublisherAccountID = strings.TrimSpace(*pubAcct)
	}
	if bind.SellerID == "" && bind.PublisherAccountID == "" {
		return PublisherBind{}, ErrPublisherScopeRequired
	}
	if bind.CustomerID == uuid.Nil {
		return PublisherBind{}, ErrPublisherScopeRequired
	}
	return bind, nil
}

func (s *Service) GetPublisherDashboard(ctx context.Context, bind PublisherBind, from, to time.Time) (PublisherDashboardDTO, error) {
	out := PublisherDashboardDTO{
		SellerID:           bind.SellerID,
		PublisherAccountID: bind.PublisherAccountID,
		From:               from.UTC().Format(time.RFC3339),
		To:                 to.UTC().Format(time.RFC3339),
		KPIs:               PublisherKPIsDTO{},
		Placements:         []PublisherPlacementDTO{},
	}
	clickhouseQuery := s.clickhouseQuery
	if clickhouseQuery == nil {
		return out, nil
	}
	clickhouseCtx, cancel := context.WithTimeout(ctx, ReportClickHouseQueryTimeout())
	defer cancel()

	needleSeller := bind.SellerID
	needlePub := bind.PublisherAccountID
	rows, err := clickhouseQuery.Query(clickhouseCtx, publisherPlacementStatsQuery, from, to, needleSeller, needlePub, needleSeller, needlePub, from, to, needleSeller, needlePub, needleSeller, needlePub)
	if err != nil {
		return PublisherDashboardDTO{}, fmt.Errorf("publisher dashboard query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var totalImpressions, totalClicks int64
	var totalRevenueMicro int64
	for rows.Next() {
		var placementID string
		var impressions, clicks uint64
		var revenueMicro int64
		if err := rows.Scan(&placementID, &impressions, &clicks, &revenueMicro); err != nil {
			return PublisherDashboardDTO{}, err
		}
		totalImpressions += int64(impressions)
		totalClicks += int64(clicks)
		totalRevenueMicro += revenueMicro
		fillRate := 0.0
		if impressions > 0 {
			fillRate = float64(clicks) / float64(impressions)
		}
		ecpmMicro := int64(0)
		if impressions > 0 {
			ecpmMicro = revenueMicro * 1000 / int64(impressions)
		}
		out.Placements = append(out.Placements, PublisherPlacementDTO{
			PlacementID:  placementID,
			Impressions:  int64(impressions),
			Clicks:       int64(clicks),
			FillRate:     fillRate,
			RevenueMicro: revenueMicro,
			EcpmMicro:    ecpmMicro,
		})
	}
	if err := rows.Err(); err != nil {
		return PublisherDashboardDTO{}, err
	}

	out.KPIs.Impressions = totalImpressions
	out.KPIs.FillRate = 0
	if totalImpressions > 0 {
		out.KPIs.FillRate = float64(totalClicks) / float64(totalImpressions)
	}
	out.KPIs.EcpmMicro = 0
	if totalImpressions > 0 {
		out.KPIs.EcpmMicro = totalRevenueMicro * 1000 / totalImpressions
	}
	out.KPIs.IVTRate = 0
	return out, nil
}

func (s *Service) ListPublisherStatements(ctx context.Context, bind PublisherBind, from, to time.Time, limit, offset int32) ([]PublisherStatementDTO, int64, error) {
	pool := s.GetPool()
	if pool == nil {
		return nil, 0, errors.New("publisher service unavailable")
	}
	scopeSeller := bind.SellerID
	scopePub := bind.PublisherAccountID

	var total int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM balance_ledger
		WHERE customer_id = $1
		 AND type = 'publisher_payout'
		 AND created_at >= $2
		 AND created_at < $3
		 AND (
		 $4 = '' OR idempotency_hash LIKE $4 || '%' OR position($4 in idempotency_hash) > 0
		 OR $5 = '' OR idempotency_hash LIKE $5 || '%' OR position($5 in idempotency_hash) > 0
		 )`,
		bind.CustomerID, from, to, scopeSeller, scopePub).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := pool.Query(ctx, `
		SELECT id, amount, created_at, COALESCE(campaign_id::text, ''), COALESCE(idempotency_hash, '')
		FROM balance_ledger
		WHERE customer_id = $1
		 AND type = 'publisher_payout'
		 AND created_at >= $2
		 AND created_at < $3
		 AND (
		 $4 = '' OR idempotency_hash LIKE $4 || '%' OR position($4 in idempotency_hash) > 0
		 OR $5 = '' OR idempotency_hash LIKE $5 || '%' OR position($5 in idempotency_hash) > 0
		 )
		ORDER BY created_at DESC, id DESC
		LIMIT $6 OFFSET $7`,
		bind.CustomerID, from, to, scopeSeller, scopePub, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]PublisherStatementDTO, 0, limit)
	for rows.Next() {
		var row PublisherStatementDTO
		var amount int64
		var createdAt time.Time
		if err := rows.Scan(&row.ID, &amount, &createdAt, &row.CampaignID, &row.IdempotencyHash); err != nil {
			return nil, 0, err
		}
		row.AmountMicro = amount
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		items = append(items, row)
	}
	return items, total, rows.Err()
}

type SupplyValidationReport struct {
	SellersJSONValid      bool     `json:"sellers_json_valid"`
	SellersChecksumSHA256 string   `json:"sellers_checksum_sha256"`
	SellersCount          int      `json:"sellers_count"`
	AdsTxtValid           bool     `json:"ads_txt_valid"`
	AdsTxtChecksumSHA256  string   `json:"ads_txt_checksum_sha256"`
	AdsTxtLineCount       int      `json:"ads_txt_line_count"`
	Issues                []string `json:"issues,omitempty"`
}

func (s *Service) ValidateSupplyFiles(ctx context.Context) (SupplyValidationReport, error) {
	out := SupplyValidationReport{Issues: []string{}}
	sellersBody, err := s.BuildSellersJSON(ctx)
	if err != nil {
		out.Issues = append(out.Issues, "sellers.json: "+err.Error())
	} else {
		sum := sha256.Sum256(sellersBody)
		out.SellersChecksumSHA256 = hex.EncodeToString(sum[:])
		var doc struct {
			Sellers []json.RawMessage `json:"sellers"`
		}
		if err := json.Unmarshal(sellersBody, &doc); err != nil {
			out.Issues = append(out.Issues, "sellers.json: invalid JSON")
		} else {
			out.SellersJSONValid = true
			out.SellersCount = len(doc.Sellers)
		}
	}

	adsTxt, err := s.BuildAdsTxt(ctx)
	if err != nil {
		out.Issues = append(out.Issues, "ads.txt: "+err.Error())
	} else {
		sum := sha256.Sum256([]byte(adsTxt))
		out.AdsTxtChecksumSHA256 = hex.EncodeToString(sum[:])
		lines := strings.Split(strings.TrimSpace(adsTxt), "\n")
		out.AdsTxtLineCount = len(lines)
		if adsTxt == "" {
			out.Issues = append(out.Issues, "ads.txt: empty export")
		} else {
			out.AdsTxtValid = true
		}
	}
	return out, nil
}

const publisherPlacementStatsQuery = `
SELECT
 placement_id,
 sum(impressions) AS impressions,
 sum(clicks) AS clicks,
 sum(revenue_micro) AS revenue_micro
FROM (
 SELECT
 placement_id,
 toUInt64(0) AS impressions,
 sum(click_count) AS clicks,
 sum(revenue_micro) AS revenue_micro
 FROM placement_stats_hourly
 WHERE hour >= ?
 AND hour < ?
 AND (
 (? != '' AND startsWith(placement_id, ?))
 OR (? != '' AND position(placement_id, ?) > 0)
 )
 GROUP BY placement_id
 UNION ALL
 SELECT
 placement_id,
 count() AS impressions,
 toUInt64(0) AS clicks,
 toInt64(0) AS revenue_micro
 FROM impressions
 WHERE created_at >= ?
 AND created_at < ?
 AND (
 (? != '' AND startsWith(placement_id, ?))
 OR (? != '' AND position(placement_id, ?) > 0)
 )
 GROUP BY placement_id
)
GROUP BY placement_id
ORDER BY impressions DESC
LIMIT 50`
