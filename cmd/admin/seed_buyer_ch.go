// Buyer dashboard ClickHouse seed: clicks, conversions, and hourly economics aligned with PG stats.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"ad-event-processor/internal/clickhouse/migrate"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/spf13/cobra"
)

const (
	defaultBuyerCHCampaignCount = 50
	seedBuyerHistoryDays        = 14
	maxBuyerClicksPerDay        = 240
	maxBuyerConversionsPerDay   = 40
)

var (
	buyerCHCampaignCount int
	buyerCHCustomerSeq   int
	buyerCHReplace       bool
)

var seedBuyerCHCmd = &cobra.Command{
	Use:   "seed-buyer-ch",
	Short: "Seed ClickHouse clicks and economics for buyer dashboard drilldown",
	Long:  "Inserts realistic traffic rows for deterministic campaigns (seq 1..N). Requires CH_DSN and PG campaign_stats from db seed-ui.",
	RunE:  runSeedBuyerCH,
}

func init() {
	seedBuyerCHCmd.Flags().IntVar(&buyerCHCampaignCount, "count", defaultBuyerCHCampaignCount, "Deterministic campaign seq to seed (1..N)")
	seedBuyerCHCmd.Flags().IntVar(&buyerCHCustomerSeq, "customer-seq", 1, "Customer seq used when resolving portfolio owner")
	seedBuyerCHCmd.Flags().BoolVar(&buyerCHReplace, "replace", true, "Delete existing CH rows for seeded campaigns before insert")
	dbCmd.AddCommand(seedBuyerCHCmd)
}

type buyerCampaignRow struct {
	id  uuid.UUID
	seq int
}

type buyerDayStats struct {
	impressions int64
	clicks      int64
	conversions int64
}

var seedBuyerTrafficSources = []string{
	"facebook", "google", "tiktok", "snapchat", "native-ads", "taboola", "email", "affiliate",
}

var seedBuyerPublishers = []string{
	"pub_northstar", "pub_velocity", "pub_summit", "pub_harbor", "pub_cedar", "pub_orbit",
}

var seedBuyerPlacements = []string{
	"lander/checkout-v2", "lander/hero-video", "lander/compare-table", "lander/quiz-funnel", "lander/app-install",
}

var seedBuyerCountries = []string{"US", "GB", "DE", "CA", "UA", "FR", "JP", "AU", "BR", "MX"}

var seedBuyerDevices = []string{"mobile", "desktop", "tablet"}

func runSeedBuyerCH(cmd *cobra.Command, args []string) error {
	if !cfg.IsClickHouseEnabled() {
		return fmt.Errorf("CH_ENABLED=0 or CH_DSN empty; set CH_ENABLED=1, CH_USE_UDS=0, and CH_DSN (TCP) before seeding ClickHouse")
	}
	if buyerCHCampaignCount < 1 {
		return fmt.Errorf("count must be >= 1")
	}

	ctx := context.Background()
	pool, err := getDB(ctx)
	if err != nil {
		return err
	}
	defer pool.Close()

	chConn, err := database.ConnectClickHouse(ctx, string(cfg.ClickHouseDSN))
	if err != nil {
		return fmt.Errorf("connect clickhouse: %w", err)
	}
	defer func() { _ = chConn.Close() }()
	if err := migrate.ApplyClickHouseMigrations(ctx, chConn); err != nil {
		return fmt.Errorf("clickhouse migrate: %w", err)
	}

	campaigns := loadBuyerSeedCampaigns()
	if len(campaigns) == 0 {
		return fmt.Errorf("no seed campaigns found; run seed_ingest_only_campaigns.sh and db seed-ui first")
	}

	campaignIDs := make([]uuid.UUID, len(campaigns))
	for i, row := range campaigns {
		campaignIDs[i] = row.id
	}

	if buyerCHReplace {
		if err := clearBuyerClickHouseRows(ctx, chConn, campaignIDs); err != nil {
			return err
		}
	}

	hasher := piihash.TestHasher()
	today := time.Now().UTC().Truncate(24 * time.Hour)
	fromDay := today.AddDate(0, 0, -(seedBuyerHistoryDays - 1))

	var clickRows, convRows, hourlyRows int
	for _, camp := range campaigns {
		statsByDay, err := loadBuyerCampaignStats(ctx, pool, camp.id, fromDay, today.Add(24*time.Hour))
		if err != nil {
			return err
		}
		for dayOffset := 0; dayOffset < seedBuyerHistoryDays; dayOffset++ {
			day := today.AddDate(0, 0, -dayOffset)
			st := statsByDay[day.Format("2006-01-02")]
			if st.clicks == 0 && st.conversions == 0 {
				imp, clk, conv := seedUiDemoDeliveryCounts(camp.seq, dayOffset)
				st = buyerDayStats{impressions: imp, clicks: clk, conversions: conv}
			}
			cInserted, vInserted, hInserted, err := seedBuyerCampaignDay(ctx, chConn, hasher, camp, day, st)
			if err != nil {
				return fmt.Errorf("campaign %s day %s: %w", camp.id, day.Format("2006-01-02"), err)
			}
			clickRows += cInserted
			convRows += vInserted
			hourlyRows += hInserted
		}
	}

	fmt.Println("Buyer ClickHouse seed complete")
	fmt.Printf("  Campaigns: %d\n", len(campaigns))
	fmt.Printf("  Click rows: %d\n", clickRows)
	fmt.Printf("  Conversion rows: %d\n", convRows)
	fmt.Printf("  Hourly rollup rows: %d\n", hourlyRows)
	fmt.Printf("  Window: last %d days\n", seedBuyerHistoryDays)
	fmt.Printf("  Demo customer: %s (%s)\n", seedCustomerUUID(buyerCHCustomerSeq), seedCustomerName(buyerCHCustomerSeq))
	return nil
}

func loadBuyerSeedCampaigns() []buyerCampaignRow {
	rows := make([]buyerCampaignRow, 0, buyerCHCampaignCount)
	for seq := 1; seq <= buyerCHCampaignCount; seq++ {
		rows = append(rows, buyerCampaignRow{id: seedCampaignUUID(seq), seq: seq})
	}
	return rows
}

func loadBuyerCampaignStats(ctx context.Context, pool interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, campaignID uuid.UUID, from, to time.Time) (map[string]buyerDayStats, error) {
	out := make(map[string]buyerDayStats, seedBuyerHistoryDays)
	rows, err := pool.Query(ctx, `
SELECT date,
 COALESCE(impressions_count, 0),
 COALESCE(clicks_count, 0),
 COALESCE(conversions_count, 0)
FROM campaign_stats
WHERE campaign_id = $1
 AND date >= $2::date
 AND date < $3::date`,
		campaignID, from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var day time.Time
		var st buyerDayStats
		if err := rows.Scan(&day, &st.impressions, &st.clicks, &st.conversions); err != nil {
			return nil, err
		}
		out[day.UTC().Format("2006-01-02")] = st
	}
	return out, rows.Err()
}

func clearBuyerClickHouseRows(ctx context.Context, conn driver.Conn, campaignIDs []uuid.UUID) error {
	tables := []string{"clicks", "conversions", "placement_stats_hourly", "cost_snapshots"}
	for _, table := range tables {
		if err := conn.Exec(ctx, fmt.Sprintf("ALTER TABLE ad_event_processor.%s DELETE WHERE campaign_id IN (?)", table), campaignIDs); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}
	return nil
}

func seedBuyerCampaignDay(
	ctx context.Context,
	conn driver.Conn,
	hasher *piihash.Hasher,
	camp buyerCampaignRow,
	day time.Time,
	st buyerDayStats,
) (clickRows, convRows, hourlyRows int, err error) {
	targetClicks := st.clicks
	if targetClicks <= 0 {
		return 0, 0, 0, nil
	}
	insertClicks := targetClicks
	if insertClicks > maxBuyerClicksPerDay {
		insertClicks = maxBuyerClicksPerDay
	}
	insertConversions := st.conversions
	if insertConversions > maxBuyerConversionsPerDay {
		insertConversions = maxBuyerConversionsPerDay
	}

	roiNumer := buyerCampaignROINumerator(camp.seq)
	totalCostMicro := buyerDayCostMicro(camp.seq, targetClicks)
	totalRevenueMicro := totalCostMicro * roiNumer / 100

	clickBatch, err := conn.PrepareBatch(ctx, `
INSERT INTO ad_event_processor.clicks (
 click_id, campaign_id, placement_id, sub1, sub2, country, device_type,
 ip_hash, ua_hash, pii_salt_version, payload, created_at, attributed_cost_micro, cost_source
)`)
	if err != nil {
		return 0, 0, 0, err
	}

	for i := int64(0); i < insertClicks; i++ {
		source := seedBuyerTrafficSources[int((camp.seq+int(i))%len(seedBuyerTrafficSources))]
		publisher := seedBuyerPublishers[int((camp.seq*3+int(i))%len(seedBuyerPublishers))]
		placement := seedBuyerPlacements[int((camp.seq*5+int(i))%len(seedBuyerPlacements))]
		country := seedBuyerCountries[int((camp.seq*7+int(i))%len(seedBuyerCountries))]
		device := seedBuyerDevices[int((camp.seq+int(i*2))%len(seedBuyerDevices))]

		hour := 8 + int(i)%14
		minute := int((i * 17) % 60)
		createdAt := time.Date(day.Year(), day.Month(), day.Day(), hour, minute, int(i%60), 0, time.UTC)

		cpcMicro := buyerCPCMicro(camp.seq, source)
		payload, _ := json.Marshal(map[string]string{
			"sub1":         source,
			"sub2":         publisher,
			"sub3":         fmt.Sprintf("feed-%d", 1+(int(i)%4)),
			"sub4":         fmt.Sprintf("creative-%d", 1+(camp.seq%12)),
			"sub5":         fmt.Sprintf("zone-%s", country),
			"country":      country,
			"device_type":  device,
			"placement_id": placement,
		})

		clickID := fmt.Sprintf("seed-%s-%s-%04d", camp.id.String()[:8], day.Format("20060102"), i)
		ipHash := piihash.FixedString16(hasher.HashIP(fmt.Sprintf("203.0.113.%d", (camp.seq+int(i))%250)))
		uaHash := piihash.FixedString16(hasher.HashUA(fmt.Sprintf("buyer-seed/%s/%d", device, i%50)))

		if err := clickBatch.Append(
			clickID,
			camp.id,
			placement,
			source,
			publisher,
			country,
			device,
			ipHash,
			uaHash,
			uint8(1),
			string(payload),
			createdAt,
			cpcMicro,
			"seed",
		); err != nil {
			_ = clickBatch.Abort()
			return 0, 0, 0, err
		}
	}
	if err := clickBatch.Send(); err != nil {
		return 0, 0, 0, err
	}
	clickRows = int(insertClicks)

	convBatch, err := conn.PrepareBatch(ctx, `
INSERT INTO ad_event_processor.conversions (
 click_id, campaign_id, placement_id, ip_hash, ua_hash, pii_salt_version, payload, created_at, device_type
)`)
	if err != nil {
		return clickRows, 0, 0, err
	}

	revenuePerConv := int64(0)
	if insertConversions > 0 {
		revenuePerConv = totalRevenueMicro / insertConversions
		if revenuePerConv < 500_000 {
			revenuePerConv = 500_000 + int64(camp.seq%7)*250_000
		}
	}

	for i := int64(0); i < insertConversions; i++ {
		source := seedBuyerTrafficSources[int((camp.seq+int(i*3))%len(seedBuyerTrafficSources))]
		publisher := seedBuyerPublishers[int((camp.seq+int(i*5))%len(seedBuyerPublishers))]
		placement := seedBuyerPlacements[int((camp.seq+int(i))%len(seedBuyerPlacements))]
		country := seedBuyerCountries[int((camp.seq+int(i*2))%len(seedBuyerCountries))]
		device := seedBuyerDevices[int((camp.seq+int(i))%len(seedBuyerDevices))]

		clickIdx := i % insertClicks
		clickID := fmt.Sprintf("seed-%s-%s-%04d", camp.id.String()[:8], day.Format("20060102"), clickIdx)
		createdAt := time.Date(day.Year(), day.Month(), day.Day(), 10+int(i)%10, int(i*11)%60, 0, 0, time.UTC)

		payload, _ := json.Marshal(map[string]any{
			"sub1":          source,
			"sub2":          publisher,
			"sub3":          fmt.Sprintf("feed-%d", 1+(int(i)%4)),
			"sub4":          fmt.Sprintf("creative-%d", 1+(camp.seq%12)),
			"sub5":          fmt.Sprintf("zone-%s", country),
			"country":       country,
			"device_type":   device,
			"placement_id":  placement,
			"revenue_micro": revenuePerConv,
			"goal_name":     buyerGoalName(camp.seq),
			"status":        "approved",
		})

		ipHash := piihash.FixedString16(hasher.HashIP(fmt.Sprintf("198.51.100.%d", (camp.seq+int(i))%200)))
		uaHash := piihash.FixedString16(hasher.HashUA(fmt.Sprintf("buyer-conv/%s/%d", device, i)))

		if err := convBatch.Append(
			clickID,
			camp.id,
			placement,
			ipHash,
			uaHash,
			uint8(1),
			string(payload),
			createdAt,
			device,
		); err != nil {
			_ = convBatch.Abort()
			return clickRows, 0, 0, err
		}
	}
	if err := convBatch.Send(); err != nil {
		return clickRows, 0, 0, err
	}
	convRows = int(insertConversions)

	hourlyBatch, err := conn.PrepareBatch(ctx, `
INSERT INTO ad_event_processor.placement_stats_hourly (
 campaign_id, placement_id, hour, spend_micro, revenue_micro, click_count, conversion_count
)`)
	if err != nil {
		return clickRows, convRows, 0, err
	}

	for h := 8; h < 22; h++ {
		hourStart := time.Date(day.Year(), day.Month(), day.Day(), h, 0, 0, 0, time.UTC)
		for sIdx, placement := range seedBuyerPlacements[:4] {
			shareNum := int64(sIdx + 2)
			shareDen := int64(14)
			spendPart := totalCostMicro * shareNum / shareDen / 14
			revPart := totalRevenueMicro * shareNum / shareDen / 14
			clkPart := uint64(math.Max(1, float64(targetClicks)*float64(shareNum)/float64(shareDen)/14))
			convPart := uint64(insertConversions * shareNum / shareDen / 14)
			if err := hourlyBatch.Append(
				camp.id,
				placement,
				hourStart,
				spendPart,
				revPart,
				clkPart,
				convPart,
			); err != nil {
				_ = hourlyBatch.Abort()
				return clickRows, convRows, 0, err
			}
			hourlyRows++
		}
	}
	if err := hourlyBatch.Send(); err != nil {
		return clickRows, convRows, 0, err
	}

	return clickRows, convRows, hourlyRows, nil
}

func buyerCampaignROINumerator(seq int) int64 {
	switch {
	case seq%19 == 0:
		return 62
	case seq%13 == 0:
		return 88
	case seq%11 == 0:
		return 175
	case seq%7 == 0:
		return 132
	default:
		return 118
	}
}

func buyerDayCostMicro(seq int, clicks int64) int64 {
	if clicks <= 0 {
		return 0
	}
	baseCPC := 950_000 + int64(seq%11)*120_000
	return baseCPC * clicks
}

func buyerCPCMicro(seq int, source string) int64 {
	base := 780_000 + int64(seq%13)*95_000
	switch source {
	case "google":
		base += 420_000
	case "facebook", "tiktok":
		base += 210_000
	case "native-ads", "taboola":
		base -= 120_000
	}
	if base < 350_000 {
		base = 350_000
	}
	return base
}

func buyerGoalName(seq int) string {
	goals := []string{"lead", "purchase", "signup", "trial", "deposit"}
	return goals[seq%len(goals)]
}
