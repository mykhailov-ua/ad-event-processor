// Ensures each customer balance covers the largest campaign budget they own (clone pre-check).
package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/spf13/cobra"
)

var seedCloneBalanceCmd = &cobra.Command{
	Use:   "seed-clone-balance",
	Short: "Raise customer balances to cover max owned campaign budget (clone E2E/dev)",
	Long:  "Sets balance = GREATEST(balance, max campaign budget_limit) per customer. Safe to re-run.",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		pool, err := getDB(ctx)
		if err != nil {
			return err
		}
		defer pool.Close()

		tag, err := ensureCustomerBalancesCoverCampaignBudgets(ctx, pool)
		if err != nil {
			return err
		}
		fmt.Printf("Clone balance seed complete (%d customers updated)\n", tag.RowsAffected())
		return nil
	},
}

func init() {
	dbCmd.AddCommand(seedCloneBalanceCmd)
}

func ensureCustomerBalancesCoverCampaignBudgets(ctx context.Context, exec interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
},
) (pgconn.CommandTag, error) {
	return exec.Exec(ctx, `
UPDATE customers c
SET balance = GREATEST(c.balance, sub.max_budget),
    updated_at = NOW()
FROM (
    SELECT customer_id, MAX(budget_limit) AS max_budget
    FROM campaigns
    WHERE customer_id IS NOT NULL
    GROUP BY customer_id
) sub
WHERE c.id = sub.customer_id`)
}
