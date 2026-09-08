// Appliance demo seed: billing invoices, landers/offers, extended campaign fields, Glory portfolio.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"
)

const (
	defaultApplianceDemoCount      = 100
	defaultApplianceGloryCustomer  = "8c66adae-da43-41d0-94fc-6a1abb54fc55"
	defaultApplianceGloryCampaigns = 35
)

var (
	applianceDemoCount          int
	applianceGloryCustomerID    string
	applianceGloryCampaignCount int
)

var seedApplianceDemoCmd = &cobra.Command{
	Use:   "seed-appliance-demo",
	Short: "Enrich PG with synthetic billing, landers, offers, and full campaign columns",
	Long:  "Run after seed-ingest-sql and db seed-ui. Seeds invoices, landers, offers, balance_ledger rows, and nullable campaign fields. Safe to re-run.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		pool, err := getDB(ctx)
		if err != nil {
			return err
		}
		defer pool.Close()

		if applianceDemoCount < 1 {
			return fmt.Errorf("count must be >= 1")
		}
		gloryID, err := uuid.Parse(applianceGloryCustomerID)
		if err != nil {
			return fmt.Errorf("invalid glory-customer-id: %w", err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		fmt.Printf("Seeding appliance demo enrichments (count=%d)...\n", applianceDemoCount)

		landers, offers, err := seedApplianceLandersOffers(ctx, tx, applianceDemoCount)
		if err != nil {
			return err
		}
		fmt.Printf("  Landers: %d, Offers: %d\n", len(landers), len(offers))

		campaignsUpdated, err := seedApplianceExtendedCampaigns(ctx, tx, applianceDemoCount, landers, offers)
		if err != nil {
			return err
		}
		fmt.Printf("  Campaigns enriched: %d\n", campaignsUpdated)

		ledgerRows, err := seedApplianceBalanceLedger(ctx, tx, applianceDemoCount)
		if err != nil {
			return err
		}
		fmt.Printf("  Balance ledger rows: %d\n", ledgerRows)

		invoices, lines, err := seedApplianceBilling(ctx, tx, applianceDemoCount)
		if err != nil {
			return err
		}
		fmt.Printf("  Invoices: %d (lines: %d)\n", invoices, lines)

		gloryAssigned, err := assignGloryPortfolio(ctx, tx, gloryID, applianceGloryCampaignCount)
		if err != nil {
			return err
		}
		fmt.Printf("  Glory campaigns assigned: %d\n", gloryAssigned)

		if err = tx.Commit(ctx); err != nil {
			return err
		}

		fmt.Println("Appliance demo seed complete")
		return nil
	},
}

func init() {
	seedApplianceDemoCmd.Flags().IntVar(&applianceDemoCount, "count", defaultApplianceDemoCount, "Deterministic entity seq ceiling (1..N)")
	seedApplianceDemoCmd.Flags().StringVar(&applianceGloryCustomerID, "glory-customer-id", defaultApplianceGloryCustomer, "Tester customer UUID to receive a campaign portfolio")
	seedApplianceDemoCmd.Flags().IntVar(&applianceGloryCampaignCount, "glory-campaigns", defaultApplianceGloryCampaigns, "Campaigns assigned to glory-customer-id")
	dbCmd.AddCommand(seedApplianceDemoCmd)
}

func seedApplianceLandersOffers(ctx context.Context, tx pgx.Tx, count int) ([]uuid.UUID, []uuid.UUID, error) {
	landerIDs := make([]uuid.UUID, 0, count)
	offerIDs := make([]uuid.UUID, 0, count)
	for seq := 1; seq <= count; seq++ {
		landerID := seedDeterministicUUID("lander", seq)
		offerID := seedDeterministicUUID("offer", seq)
		landerName := fmt.Sprintf("%s — %s", seedCampaignName(seq), seedCampaignFlightLabels[(seq-1)%len(seedCampaignFlightLabels)])
		offerName := fmt.Sprintf("%s offer", seedCampaignName(seq))

		_, err := tx.Exec(ctx, `
INSERT INTO landers (id, name, url)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url`,
			pgtype.UUID{Bytes: landerID, Valid: true},
			landerName,
			fmt.Sprintf("https://lp.horizon-media.io/%d?cid={click_id}", seq),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("lander seq=%d: %w", seq, err)
		}

		_, err = tx.Exec(ctx, `
INSERT INTO offers (id, name, url)
VALUES ($1, $2, $3)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, url = EXCLUDED.url`,
			pgtype.UUID{Bytes: offerID, Valid: true},
			offerName,
			fmt.Sprintf("https://offers.horizon-media.io/%d?cid={click_id}", seq),
		)
		if err != nil {
			return nil, nil, fmt.Errorf("offer seq=%d: %w", seq, err)
		}

		landerIDs = append(landerIDs, landerID)
		offerIDs = append(offerIDs, offerID)
	}
	return landerIDs, offerIDs, nil
}

func seedApplianceExtendedCampaigns(
	ctx context.Context,
	tx pgx.Tx,
	count int,
	landers []uuid.UUID,
	offers []uuid.UUID,
) (int, error) {
	now := time.Now().UTC()
	var updated int
	for seq := 1; seq <= count; seq++ {
		campID := seedCampaignUUID(seq)
		landerIdx := (seq - 1) % len(landers)
		offerIdx := (seq - 1) % len(offers)
		startAt := now.AddDate(0, 0, -(seq % 45))
		endAt := now.AddDate(0, 3+(seq%6), seq%28)
		daypart := []int16{8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19}
		creativePayload, _ := json.Marshal(map[string]string{
			"headline": seedCampaignName(seq),
			"cta":      seedCampaignGoalLabels[(seq-1)%len(seedCampaignGoalLabels)],
			"subhead":  seedCampaignFlightLabels[(seq-1)%len(seedCampaignFlightLabels)],
		})
		clickQueryPayload, _ := json.Marshal(map[string]string{
			"utm_source":   seedBuyerTrafficSources[(seq-1)%len(seedBuyerTrafficSources)],
			"utm_medium":   "cpc",
			"utm_campaign": fmt.Sprintf("camp-%03d", seq),
			"sub_id":       fmt.Sprintf("pub-%02d", seq%24),
			"click_id":     "{click_id}",
		})
		fraudPass := uint8(22 + (seq % 38))
		fraudSuspect := fraudPass + uint8(12+(seq%8))
		fraudIVT := fraudSuspect + uint8(10+(seq%6))
		fraudBlock := fraudIVT + uint8(8+(seq%5))
		if fraudBlock > 98 {
			fraudBlock = 98
		}
		attestationModes := []string{"off", "light", "strict"}
		reviewActions := []string{"safe_page", "block", "passthrough"}
		connPolicies := []string{"block_vpn_hosting", "mobile_only", "residential_only"}
		clickDeliveries := []string{"redirect", "proxy"}

		tag, err := tx.Exec(ctx, `
UPDATE campaigns
SET start_at = $2,
    end_at = $3,
    daypart_hours = $4,
    safe_page_url = $5,
    safe_page_enabled = $6,
    referrer_filter = $7,
    creative_payload = $8::jsonb,
    target_url = $9,
    silent_reject_enabled = $10,
    fraud_threshold_pass = $11,
    fraud_threshold_suspect = $12,
    fraud_threshold_ivt = $13,
    fraud_threshold_block = $14,
    behavior_flags = $15,
    reserve_micro = $16,
    click_query_params = $17::jsonb,
    link_signing_enabled = $18,
    link_signing_ttl_sec = $19,
    attestation_enabled = $20,
    attestation_mode = $21,
    attestation_ttl_sec = $22,
    cidr_block_enabled = $23,
    proxy_vpn_block_enabled = $24,
    dmr_enabled = $25,
    moderator_intel_enabled = $26,
    review_traffic_action = $27,
    tls_fingerprint_block_enabled = $28,
    conn_type_policy = $29,
    click_delivery = $30,
    social_in_app_enabled = $31,
    canvas_retest_enabled = $32,
    cgnat_ip_policy_enabled = $33,
    accept_lang_geo_enabled = $34,
    json_serialization_enabled = $35,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
			pgtype.UUID{Bytes: campID, Valid: true},
			startAt,
			endAt,
			daypart,
			fmt.Sprintf("https://safe.horizon-media.io/%d", seq),
			seq%11 == 0,
			"google.com|facebook.com|tiktok.com",
			string(creativePayload),
			fmt.Sprintf("https://trk.horizon-media.io/%d?cid={click_id}", seq),
			seq%13 == 0,
			fraudPass,
			fraudSuspect,
			fraudIVT,
			fraudBlock,
			uint32(seq%17),
			int64(250_000+(seq%9)*75_000),
			string(clickQueryPayload),
			seq%5 == 0,
			int32(300+(seq%7)*60),
			seq%4 != 0,
			attestationModes[seq%len(attestationModes)],
			int32(120+(seq%5)*30),
			seq%6 == 0,
			seq%7 == 0,
			seq%8 == 0,
			seq%9 == 0,
			reviewActions[seq%len(reviewActions)],
			seq%10 == 0,
			connPolicies[seq%len(connPolicies)],
			clickDeliveries[seq%len(clickDeliveries)],
			seq%3 == 0,
			seq%12 == 0,
			seq%14 == 0,
			seq%15 == 0,
			seq%16 == 0,
		)
		if err != nil {
			return updated, fmt.Errorf("campaign enrich seq=%d: %w", seq, err)
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		updated++

		flowID := seedDeterministicUUID("flow", seq)
		pathPayload := fmt.Sprintf(
			`[{"weight":100,"filters":{"countries":["US","GB"]},"landers":[{"lander_id":"%s","weight":100}],"offers":[{"offer_id":"%s","weight":100}]}]`,
			landers[landerIdx].String(),
			offers[offerIdx].String(),
		)
		_, err = tx.Exec(ctx, `
INSERT INTO flows (id, name, paths)
VALUES ($1, $2, $3::jsonb)
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, paths = EXCLUDED.paths`,
			pgtype.UUID{Bytes: flowID, Valid: true},
			fmt.Sprintf("Flow %s", seedCampaignName(seq)),
			pathPayload,
		)
		if err != nil {
			return updated, fmt.Errorf("flow seq=%d: %w", seq, err)
		}

		_, err = tx.Exec(ctx, `
UPDATE campaigns SET flow_id = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`,
			pgtype.UUID{Bytes: campID, Valid: true},
			pgtype.UUID{Bytes: flowID, Valid: true},
		)
		if err != nil {
			return updated, fmt.Errorf("campaign flow seq=%d: %w", seq, err)
		}
	}
	return updated, nil
}

func seedApplianceBalanceLedger(ctx context.Context, tx pgx.Tx, count int) (int, error) {
	now := time.Now().UTC()
	var rows int
	for seq := 1; seq <= count; seq++ {
		campID := seedCampaignUUID(seq)
		customerID := seedCustomerUUID((seq-1)%count + 1)
		entries := []struct {
			amount int64
			typ    string
			suffix string
		}{
			{-int64(2_500_000 + seq*17_431), "FEE", "fee"},
			{-int64(8_000_000 + seq*53_221), "rtb_cost", "rtb"},
			{int64(12_000_000 + seq*41_009), "RECONCILIATION_ADJUST", "adj"},
		}
		for _, entry := range entries {
			hash := fmt.Sprintf("seed-appliance-%s-%s-%s", campID.String(), entry.suffix, customerID.String())
			_, err := tx.Exec(ctx, `
INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (idempotency_hash) DO NOTHING`,
				pgtype.UUID{Bytes: customerID, Valid: true},
				pgtype.UUID{Bytes: campID, Valid: true},
				entry.amount,
				entry.typ,
				hash,
				now.Add(-time.Duration(seq%72)*time.Hour),
			)
			if err != nil {
				return rows, fmt.Errorf("balance_ledger seq=%d: %w", seq, err)
			}
			rows++
		}
	}
	return rows, nil
}

func seedApplianceBilling(ctx context.Context, tx pgx.Tx, count int) (int, int, error) {
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	var invoices, lines int
	customerCap := count
	if customerCap > 30 {
		customerCap = 30
	}
	for seq := 1; seq <= customerCap; seq++ {
		customerID := seedCustomerUUID(seq)
		for monthOffset := range 4 {
			billingMonth := monthStart.AddDate(0, -monthOffset, 0)
			invoiceID := seedDeterministicUUID("invoice", seq*10+monthOffset)
			subtotal := int64(4_800_000_000 + int64(seq*monthOffset+1)*127_000_000)
			tax := subtotal * int64(8+seq%5) / 100
			total := subtotal + tax
			status := "FINALIZED"
			if monthOffset == 0 && seq%5 == 0 {
				status = "VOID"
			}

			_, err := tx.Exec(ctx, `
INSERT INTO billing.invoices (
  id, customer_id, billing_month, subtotal_micro, tax_micro, total_micro,
  currency, tax_scheme, tax_rate_bps, ledger_sum_micro, status
)
VALUES ($1, $2, $3::date, $4, $5, $6, 'USD', 'NONE', $7, $4, $8::billing.invoice_status)
ON CONFLICT (customer_id, billing_month) DO UPDATE SET
  subtotal_micro = EXCLUDED.subtotal_micro,
  tax_micro = EXCLUDED.tax_micro,
  total_micro = EXCLUDED.total_micro,
  ledger_sum_micro = EXCLUDED.ledger_sum_micro,
  status = EXCLUDED.status`,
				pgtype.UUID{Bytes: invoiceID, Valid: true},
				pgtype.UUID{Bytes: customerID, Valid: true},
				billingMonth,
				subtotal,
				tax,
				total,
				800+(seq%5)*50,
				status,
			)
			if err != nil {
				return invoices, lines, fmt.Errorf("invoice customer=%d month=%d: %w", seq, monthOffset, err)
			}
			invoices++

			lineTypes := []struct {
				typ    string
				amount int64
				count  int
			}{
				{"FEE", subtotal * 7 / 10, 40 + seq},
				{"rtb_cost", subtotal * 2 / 10, 12 + seq%7},
				{"RECONCILIATION_ADJUST", subtotal / 10, 3 + seq%4},
			}
			for _, line := range lineTypes {
				_, err = tx.Exec(ctx, `
INSERT INTO billing.invoice_lines (invoice_id, ledger_type, amount_micro, entry_count)
SELECT $1, $2, $3, $4
WHERE NOT EXISTS (
  SELECT 1 FROM billing.invoice_lines
  WHERE invoice_id = $1 AND ledger_type = $2 AND amount_micro = $3
)`,
					pgtype.UUID{Bytes: invoiceID, Valid: true},
					line.typ,
					line.amount,
					line.count,
				)
				if err != nil {
					return invoices, lines, fmt.Errorf("invoice line: %w", err)
				}
				lines++
			}
		}
	}
	return invoices, lines, nil
}

func assignGloryPortfolio(ctx context.Context, tx pgx.Tx, gloryID uuid.UUID, campaignCount int) (int, error) {
	_, err := tx.Exec(ctx, `
UPDATE customers
SET name = 'Glory Trading',
    balance = 32500000000,
    currency = 'USD',
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
		pgtype.UUID{Bytes: gloryID, Valid: true},
	)
	if err != nil {
		return 0, err
	}

	tag, err := tx.Exec(ctx, `
UPDATE campaigns
SET customer_id = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE id IN (
  SELECT id FROM campaigns
  WHERE deleted_at IS NULL
  ORDER BY id
  LIMIT $2
)`,
		pgtype.UUID{Bytes: gloryID, Valid: true},
		campaignCount,
	)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}
