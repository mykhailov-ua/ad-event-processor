package costsync

import (
	"context"
	"fmt"
	"strings"
	"time"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type FXRateCache map[string]int64

func (c *CurrencyConverter) PrepareFXCache(ctx context.Context, lines []CostLine, rateDate time.Time) (FXRateCache, error) {
	cache := make(FXRateCache)
	curSet := make(map[string]struct{})
	for _, line := range lines {
		cur := normalizeCurrency(line.Currency)
		if cur == "" || cur == "USD" {
			continue
		}
		curSet[cur] = struct{}{}
	}
	if len(curSet) == 0 {
		return cache, nil
	}

	currencies := make([]string, 0, len(curSet))
	for cur := range curSet {
		currencies = append(currencies, cur)
	}

	if c.pool != nil {
		q := db.New(c.pool)
		rows, err := q.ListECBRatesForDate(ctx, db.ListECBRatesForDateParams{
			RateDate: pgtype.Date{Time: rateDate, Valid: true},
			Column2:  currencies,
		})
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.UsdPerUnitMicro > 0 {
				cache[row.Currency] = row.UsdPerUnitMicro
			}
		}
	}

	missing := make([]string, 0)
	for _, cur := range currencies {
		if cache[cur] <= 0 {
			missing = append(missing, cur)
		}
	}
	if len(missing) == 0 {
		return cache, nil
	}

	rates, err := c.fetchECBRates(ctx)
	if err != nil {
		return nil, err
	}
	eurPerUSD, ok := rates["USD"]
	if !ok || eurPerUSD <= 0 {
		return nil, fmt.Errorf("ecb: missing USD rate")
	}

	if c.pool != nil {
		q := db.New(c.pool)
		for _, currency := range missing {
			micro, err := usdPerUnitMicroFromECB(currency, rates, eurPerUSD)
			if err != nil {
				return nil, err
			}
			cache[currency] = micro
			_ = q.UpsertECBRate(ctx, db.UpsertECBRateParams{
				RateDate:        pgtype.Date{Time: rateDate, Valid: true},
				Currency:        currency,
				UsdPerUnitMicro: micro,
			})
		}
	} else {
		for _, currency := range missing {
			micro, err := usdPerUnitMicroFromECB(currency, rates, eurPerUSD)
			if err != nil {
				return nil, err
			}
			cache[currency] = micro
		}
	}
	return cache, nil
}

func usdPerUnitMicroFromECB(currency string, rates map[string]float64, eurPerUSD float64) (int64, error) {
	if currency == "EUR" {
		return rateToMicro(1.0 / eurPerUSD)
	}
	eurPerUnit, ok := rates[currency]
	if !ok || eurPerUnit <= 0 {
		return 0, fmt.Errorf("ecb: unknown currency %s", currency)
	}
	return rateToMicro(eurPerUnit / eurPerUSD)
}

func (c *CurrencyConverter) ToUSDMicroCached(amountMicro int64, currency string, cache FXRateCache) (int64, error) {
	cur := normalizeCurrency(currency)
	if cur == "" || cur == "USD" {
		return amountMicro, nil
	}
	usdPerUnit, ok := cache[cur]
	if !ok || usdPerUnit <= 0 {
		return 0, fmt.Errorf("fx cache missing currency %s", cur)
	}
	converted := (amountMicro * usdPerUnit) / microUnit
	if converted < 0 {
		return converted, nil
	}
	return converted, nil
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func insertCampaignCostsBatch(ctx context.Context, tx pgx.Tx, lines []CostLine, usdAmounts []int64) (int, error) {
	n := len(lines)
	if n == 0 {
		return 0, nil
	}
	if len(usdAmounts) != n {
		return 0, fmt.Errorf("usd amounts length mismatch")
	}

	customerIDs := make([]uuid.UUID, n)
	campaignIDs := make([]uuid.UUID, n)
	costDates := make([]time.Time, n)
	networks := make([]string, n)
	placementIDs := make([]string, n)
	adsetIDs := make([]string, n)
	adIDs := make([]string, n)
	lineTypes := make([]string, n)
	amountMicros := make([]int64, n)
	currencies := make([]string, n)
	amountUsdMicros := make([]int64, n)
	ingestKeys := make([]string, n)

	for i, line := range lines {
		customerIDs[i] = line.CustomerID
		campaignIDs[i] = line.CampaignID
		costDates[i] = line.Date
		networks[i] = line.Network
		placementIDs[i] = line.PlacementID
		adsetIDs[i] = line.AdsetID
		adIDs[i] = line.AdID
		lineTypes[i] = string(line.LineType)
		amountMicros[i] = line.AmountMicro
		currencies[i] = line.Currency
		amountUsdMicros[i] = usdAmounts[i]
		ingestKeys[i] = IngestKey(line.CustomerID, line.CampaignID, line.Date, line.Network, line.PlacementID, line.LineType)
	}

	rows, err := tx.Query(ctx, `
		INSERT INTO campaign_costs (
			customer_id, campaign_id, cost_date, network, placement_id,
			adset_id, ad_id, line_type, amount_micro, currency, amount_usd_micro, ingest_key
		)
		SELECT
			v.customer_id, v.campaign_id, v.cost_date, v.network, v.placement_id,
			v.adset_id, v.ad_id, v.line_type, v.amount_micro, v.currency, v.amount_usd_micro, v.ingest_key
		FROM unnest(
			$1::uuid[], $2::uuid[], $3::date[], $4::text[], $5::text[],
			$6::text[], $7::text[], $8::text[], $9::bigint[], $10::text[], $11::bigint[], $12::text[]
		) AS v(
			customer_id, campaign_id, cost_date, network, placement_id,
			adset_id, ad_id, line_type, amount_micro, currency, amount_usd_micro, ingest_key
		)
		ON CONFLICT (ingest_key) DO NOTHING
		RETURNING 1`,
		customerIDs, campaignIDs, costDates, networks, placementIDs,
		adsetIDs, adIDs, lineTypes, amountMicros, currencies, amountUsdMicros, ingestKeys,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	imported := 0
	for rows.Next() {
		var one int
		if err := rows.Scan(&one); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, rows.Err()
}
