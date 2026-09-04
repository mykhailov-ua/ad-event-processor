// Campaign list UX seed: owners, countries, margin ledger, multi-customer spread.
package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/identity"
	authdb "ad-event-processor/internal/identity/db"
	"ad-event-processor/internal/ledger"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"
)

const defaultCampaignListUXCount = 120

var campaignListUXCount int

var seedCampaignListUXCmd = &cobra.Command{
	Use:   "seed-campaign-list-ux",
	Short: "Enrich seeded campaigns for admin list UX (owners, countries, margin breach, customers)",
	Long:  "Upserts team users, assigns owners and target_countries, spreads campaigns across customers, and seeds balance_ledger rows for margin breach badges. Run after seed-ui.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		pool, err := getDB(ctx)
		if err != nil {
			return err
		}
		defer pool.Close()

		if campaignListUXCount < 1 {
			return fmt.Errorf("count must be >= 1")
		}

		hasher, err := identity.NewPasswordHasher(
			uint32(cfg.Argon2Memory),
			uint32(cfg.Argon2Iterations),
			uint8(cfg.Argon2Parallelism),
		)
		if err != nil {
			return err
		}
		passwordHash, err := hasher.HashPassword("Password123!")
		if err != nil {
			return err
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

		authQueries := authdb.New(tx)
		teamUserIDs, err := ensureSeedTeamUsers(ctx, authQueries, passwordHash, seedCustomerUUID(1))
		if err != nil {
			return err
		}

		fmt.Printf("Seeding campaign list UX for up to %d campaigns...\n", campaignListUXCount)
		var updated int
		now := time.Now().UTC()

		for seq := 1; seq <= campaignListUXCount; seq++ {
			campID := seedCampaignUUID(seq)
			customerID := seedCampaignListUXCustomerID(seq)
			ownerID := teamUserIDs[(seq-1)%len(teamUserIDs)]
			countries := seedUiDemoTargetCountries(seq)

			tag, execErr := tx.Exec(ctx, `
UPDATE campaigns
SET customer_id = $2,
    owner_user_id = $3,
    target_countries = $4,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
				pgtype.UUID{Bytes: campID, Valid: true},
				pgtype.UUID{Bytes: customerID, Valid: true},
				pgtype.UUID{Bytes: ownerID, Valid: true},
				countries,
			)
			if execErr != nil {
				return fmt.Errorf("update campaign seq=%d: %w", seq, execErr)
			}
			if tag.RowsAffected() == 0 {
				continue
			}
			updated++

			if seq%29 != 0 {
				continue
			}
			if err = seedCampaignMarginBreachLedger(ctx, tx, campID, customerID, seq, now); err != nil {
				return err
			}
		}

		if err = tx.Commit(ctx); err != nil {
			return err
		}

		if updated == 0 {
			return fmt.Errorf("no campaigns updated; run seed_ingest_only_campaigns.sh and db seed-ui first")
		}

		fmt.Println("Campaign list UX seed complete")
		fmt.Printf("  Campaigns updated: %d\n", updated)
		fmt.Printf("  Team users: %d\n", len(teamUserIDs))
		fmt.Printf("  Margin breach ledger rows: ~%d campaigns (seq %% 29)\n", updated/29)
		return nil
	},
}

func init() {
	seedCampaignListUXCmd.Flags().IntVar(&campaignListUXCount, "count", defaultCampaignListUXCount, "Max deterministic campaign seq to enrich (1..N)")
	dbCmd.AddCommand(seedCampaignListUXCmd)
}

func seedCampaignListUXCustomerID(seq int) uuid.UUID {
	switch {
	case seq <= 90:
		return seedCustomerUUID(1)
	case seq <= 105:
		return seedCustomerUUID(2)
	default:
		return seedCustomerUUID(3)
	}
}

func seedTeamUserEmail(seq int) string {
	return fmt.Sprintf("campaign-list-owner-%02d@horizon-media.test", seq)
}

func ensureSeedTeamUsers(ctx context.Context, q *authdb.Queries, passwordHash string, customerID uuid.UUID) ([]uuid.UUID, error) {
	const teamSize = 8
	ids := make([]uuid.UUID, 0, teamSize)
	for seq := 1; seq <= teamSize; seq++ {
		email := seedTeamUserEmail(seq)
		user, err := q.GetUserByEmail(ctx, email)
		var userID uuid.UUID
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("lookup team user %s: %w", email, err)
			}
			created, createErr := q.CreateUser(ctx, authdb.CreateUserParams{
				Email:        email,
				PasswordHash: passwordHash,
				Role:         "advertiser",
				CustomerID:   pgtype.UUID{Bytes: customerID, Valid: true},
			})
			if createErr != nil {
				return nil, fmt.Errorf("create team user %s: %w", email, createErr)
			}
			userID = postgresUUIDToGoogleUUID(created.ID)
		} else {
			userID = postgresUUIDToGoogleUUID(user.ID)
		}
		ids = append(ids, userID)
	}
	return ids, nil
}

func seedCampaignMarginBreachLedger(
	ctx context.Context,
	tx pgx.Tx,
	campaignID, customerID uuid.UUID,
	seq int,
	now time.Time,
) error {
	advertiserSpendMicro := int64(8_000_000 + int64(seq%17)*500_000)
	rtbCostMicro := ledger.CostOverRevenueLimitMicro(advertiserSpendMicro, 500) + int64(250_000+(seq%5)*100_000)
	operatorMarginMicro := int64(400_000 + int64(seq%7)*50_000)
	createdAt := now.Add(-time.Duration(seq%72) * time.Hour)

	entries := []struct {
		amount int64
		typ    string
		suffix string
	}{
		{-advertiserSpendMicro, "FEE", "fee"},
		{rtbCostMicro, "rtb_cost", "rtb"},
		{operatorMarginMicro, "operator_margin", "margin"},
	}
	for _, entry := range entries {
		hash := fmt.Sprintf("seed-list-ux-%s-%s", campaignID.String(), entry.suffix)
		_, err := tx.Exec(ctx, `
INSERT INTO balance_ledger (customer_id, campaign_id, amount, type, idempotency_hash, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (idempotency_hash) DO UPDATE SET
  amount = EXCLUDED.amount,
  created_at = EXCLUDED.created_at`,
			pgtype.UUID{Bytes: customerID, Valid: true},
			pgtype.UUID{Bytes: campaignID, Valid: true},
			entry.amount,
			entry.typ,
			hash,
			createdAt,
		)
		if err != nil {
			return fmt.Errorf("ledger seq=%d type=%s: %w", seq, entry.typ, err)
		}
	}
	return nil
}
