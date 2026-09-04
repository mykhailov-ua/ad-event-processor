// UI demo seed: varied campaign names, spend, and campaign_stats for admin charts (PG-only dev).
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"
)

const defaultUIDemoCampaignCount = 50

var uiDemoCampaignCount int

var seedUIDemoCmd = &cobra.Command{
	Use:   "seed-ui",
	Short: "Seed campaign UI demo data (names, spend, delivery stats for charts)",
	Long:  "Upserts delivery stats and budget usage for deterministic seed campaigns (seq 1..N). Safe to re-run.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		pool, err := getDB(ctx)
		if err != nil {
			return err
		}
		defer pool.Close()

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		if uiDemoCampaignCount < 1 {
			return fmt.Errorf("count must be >= 1")
		}

		fmt.Printf("Seeding UI demo data for up to %d campaigns...\n", uiDemoCampaignCount)
		today := time.Now().UTC().Truncate(24 * time.Hour)
		var updated, skipped int

		for seq := 1; seq <= uiDemoCampaignCount; seq++ {
			campID := seedCampaignUUID(seq)
			budgetLimit := seedUiDemoBudgetMicro(seq)
			currentSpend := seedUiDemoSpendMicro(seq, budgetLimit)
			status := seedUiDemoStatus(seq)
			pacing := seedUiDemoPacing(seq)
			timezone := seedUiDemoTimezone(seq)

			tag, err := tx.Exec(ctx, `
UPDATE campaigns
SET name = $2,
    budget_limit = $3,
    current_spend = $4,
    status = $5::campaign_status_type,
    pacing_mode = $6::pacing_mode_type,
    timezone = $7,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
				pgtype.UUID{Bytes: campID, Valid: true},
				seedCampaignName(seq),
				budgetLimit,
				currentSpend,
				status,
				pacing,
				timezone,
			)
			if err != nil {
				return fmt.Errorf("update campaign %d: %w", seq, err)
			}
			if tag.RowsAffected() == 0 {
				skipped++
				continue
			}
			updated++

			for dayOffset := 0; dayOffset < 14; dayOffset++ {
				statsDate := today.AddDate(0, 0, -dayOffset)
				imp, clk, conv := seedUiDemoDeliveryCounts(seq, dayOffset)
				_, err = tx.Exec(ctx, `
INSERT INTO campaign_stats (campaign_id, date, impressions_count, clicks_count, conversions_count)
VALUES ($1, $2::date, $3, $4, $5)
ON CONFLICT (campaign_id, date) DO UPDATE SET
  impressions_count = EXCLUDED.impressions_count,
  clicks_count = EXCLUDED.clicks_count,
  conversions_count = EXCLUDED.conversions_count`,
					pgtype.UUID{Bytes: campID, Valid: true},
					statsDate,
					imp,
					clk,
					conv,
				)
				if err != nil {
					return fmt.Errorf("campaign_stats seq=%d day=%d: %w", seq, dayOffset, err)
				}
			}
		}

		if err = tx.Commit(ctx); err != nil {
			return err
		}

		if updated == 0 {
			return fmt.Errorf("no campaigns updated; run seed_ingest_only_campaigns.sh or db seed first")
		}

		fmt.Println("UI demo seed complete")
		fmt.Printf("  Campaigns updated: %d\n", updated)
		if skipped > 0 {
			fmt.Printf("  Campaigns skipped (missing): %d\n", skipped)
		}
		fmt.Println("  Stats window: last 14 days (includes today)")
		fmt.Println("  Open Campaigns in admin UI and click Budget used to view charts")
		return nil
	},
}

func init() {
	seedUIDemoCmd.Flags().IntVar(&uiDemoCampaignCount, "count", defaultUIDemoCampaignCount, "Max deterministic campaign seq to upsert (1..N)")
}

func seedUiDemoBudgetMicro(seq int) int64 {
	base := int64(3_800_000_000)
	spread := int64(19_400_000_000)
	step := int64(1_337_421)
	return base + ((int64(seq) * step) % spread) + int64(seq%13)*271_829
}

func seedUiDemoSpendMicro(seq int, budgetLimit int64) int64 {
	pcts := []int64{0, 4, 11, 23, 38, 57, 72, 84, 91, 97, 99}
	pct := pcts[seq%len(pcts)]
	spend := budgetLimit * pct / 100
	if seq%17 == 0 && spend > 0 {
		spend = spend * 93 / 100
	}
	return spend
}

func seedUiDemoStatus(seq int) string {
	switch {
	case seq%23 == 0:
		return "ARCHIVED"
	case seq%19 == 0:
		return "DELETED"
	case seq%13 == 0:
		return "EXHAUSTED"
	case seq%7 == 0:
		return "PAUSED"
	default:
		return "ACTIVE"
	}
}

func seedUiDemoPacing(seq int) string {
	if seq%3 == 0 {
		return "EVEN"
	}
	return "ASAP"
}

func seedUiDemoTimezone(seq int) string {
	zones := []string{"UTC", "America/New_York", "Europe/Berlin", "Asia/Tokyo", "Australia/Sydney"}
	return zones[seq%len(zones)]
}

func seedUiDemoDeliveryCounts(seq int, dayOffset int) (impressions, clicks, conversions int64) {
	seed := uint64(seq*1_000_003 + dayOffset*97_821)
	seed ^= seed << 13
	seed ^= seed >> 7
	seed ^= seed << 17

	impressions = int64(1_200 + (seed % 418_000) + uint64(dayOffset%6)*19_317)
	clicks = int64(18 + (seed % 7_400) + uint64(seq%17)*113)
	conversions = int64(1 + (seed % 420) + uint64(seq%9))

	if seq%9 == 0 {
		impressions = impressions * 2 / 5
		clicks = clicks / 2
	}
	if seq%11 == 0 {
		impressions = impressions * 3 / 2
		clicks = clicks * 5 / 4
	}
	if dayOffset == 0 && seq%5 == 0 {
		clicks = clicks * 3 / 2
	}
	return impressions, clicks, conversions
}
