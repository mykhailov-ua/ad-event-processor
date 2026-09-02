// Buyer dashboard PG seed: group deterministic campaigns under one demo customer.
package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"
)

const defaultBuyerPGCampaignCount = 50

var (
	buyerPGCampaignCount int
	buyerPGCustomerSeq   int
)

var seedBuyerPGCmd = &cobra.Command{
	Use:   "seed-buyer-pg",
	Short: "Assign seeded campaigns to one customer for buyer dashboard portfolio view",
	Long:  "Points deterministic seed campaigns (seq 1..N) at one customer (default seq 1). Safe to re-run.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		pool, err := getDB(ctx)
		if err != nil {
			return err
		}
		defer pool.Close()

		if buyerPGCampaignCount < 1 {
			return fmt.Errorf("count must be >= 1")
		}
		if buyerPGCustomerSeq < 1 {
			return fmt.Errorf("customer-seq must be >= 1")
		}

		customerID := seedCustomerUUID(buyerPGCustomerSeq)
		customerName := seedCustomerName(buyerPGCustomerSeq)

		tx, err := pool.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		tag, err := tx.Exec(ctx, `
INSERT INTO customers (id, name, balance, currency, allowed_overdraft)
VALUES ($1, $2, $3, 'USD', 0)
ON CONFLICT (id) DO UPDATE SET
  name = EXCLUDED.name,
  balance = GREATEST(customers.balance, EXCLUDED.balance),
  updated_at = CURRENT_TIMESTAMP`,
			pgtype.UUID{Bytes: customerID, Valid: true},
			customerName,
			seedCustomerBalanceMicro(buyerPGCustomerSeq),
		)
		if err != nil {
			return fmt.Errorf("upsert demo customer: %w", err)
		}
		_ = tag

		var assigned int64
		for seq := 1; seq <= buyerPGCampaignCount; seq++ {
			campID := seedCampaignUUID(seq)
			tag, err = tx.Exec(ctx, `
UPDATE campaigns
SET customer_id = $2,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1`,
				pgtype.UUID{Bytes: campID, Valid: true},
				pgtype.UUID{Bytes: customerID, Valid: true},
			)
			if err != nil {
				return fmt.Errorf("assign campaign seq=%d: %w", seq, err)
			}
			assigned += tag.RowsAffected()
		}

		if err = tx.Commit(ctx); err != nil {
			return err
		}

		fmt.Println("Buyer PG seed complete")
		fmt.Printf("  Customer: %s (%s)\n", customerID, customerName)
		fmt.Printf("  Campaigns assigned: %d (seq 1..%d)\n", assigned, buyerPGCampaignCount)
		if assigned == 0 {
			return fmt.Errorf("no campaigns assigned; run seed_ingest_only_campaigns.sh first")
		}
		return nil
	},
}

func init() {
	seedBuyerPGCmd.Flags().IntVar(&buyerPGCampaignCount, "count", defaultBuyerPGCampaignCount, "Deterministic campaign seq to assign (1..N)")
	seedBuyerPGCmd.Flags().IntVar(&buyerPGCustomerSeq, "customer-seq", 1, "Deterministic customer seq that owns the portfolio")
	dbCmd.AddCommand(seedBuyerPGCmd)
}
