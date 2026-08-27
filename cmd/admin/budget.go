package main

import (
	"context"
	"fmt"

	ingestdb "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/spf13/cobra"
)

var budgetCmd = &cobra.Command{
	Use:   "budget",
	Short: "Manage and reset campaign budget caches",
}

var budgetResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset the Redis budget cache and spend limits for a campaign",
	RunE: func(cmd *cobra.Command, args []string) error {
		campIDStr, _ := cmd.Flags().GetString("campaign-id")
		resetSpend, _ := cmd.Flags().GetBool("reset-db-spend")

		campaignID, err := uuid.Parse(campIDStr)
		if err != nil {
			return fmt.Errorf("invalid campaign-id UUID: %w", err)
		}

		ctx := context.Background()
		pool, err := getDB(ctx)
		if err != nil {
			return err
		}
		defer pool.Close()

		queries := ingestdb.New(pool)
		camp, err := queries.GetCampaign(ctx, pgtype.UUID{Bytes: campaignID, Valid: true})
		if err != nil {
			return fmt.Errorf("campaign not found in database: %w", err)
		}

		_ = pgUUIDToGoogleUUID(camp.CustomerID)

		redisClients, sharder, err := getRedisShards(ctx)
		if err != nil {
			return err
		}
		defer func() {
			for _, redisClient := range redisClients {
				_ = redisClient.Close()
			}
		}()

		shardIdx := sharder.GetShard(campaignID)
		redisClient := redisClients[shardIdx]

		fmt.Printf("Campaign %s maps to Redis Shard %d/%d\n", campaignID, shardIdx, len(redisClients))

		budgetKey := fmt.Sprintf("budget:campaign:%s", campaignID)
		syncKey := fmt.Sprintf("budget:sync:campaign:%s", campaignID)

		budgetDelCount, err := redisClient.Del(ctx, budgetKey).Result()
		if err != nil {
			return fmt.Errorf("failed to delete remaining budget cache: %w", err)
		}

		syncDelCount, err := redisClient.Del(ctx, syncKey).Result()
		if err != nil {
			return fmt.Errorf("failed to delete campaign sync accumulator: %w", err)
		}

		dirtyRemoveCount, err := redisClient.SRem(ctx, "budget:dirty_campaigns", campaignID.String()).Result()
		if err != nil {
			return fmt.Errorf("failed to remove campaign from dirty set: %w", err)
		}

		fmt.Printf("Cleared Redis cache:\n DEL %s (%d)\n DEL %s (%d)\n SREM budget:dirty_campaigns %s (%d)\n",
			budgetKey, budgetDelCount, syncKey, syncDelCount, campaignID, dirtyRemoveCount)

		if resetSpend {
			fmt.Println("Resetting database current_spend to 0...")
			_, err = pool.Exec(ctx, "UPDATE campaigns SET current_spend = 0, status = 'ACTIVE', updated_at = NOW() WHERE id = $1", pgtype.UUID{Bytes: campaignID, Valid: true})
			if err != nil {
				return fmt.Errorf("failed to update current_spend in DB: %w", err)
			}
			fmt.Println("PostgreSQL campaign current_spend successfully reset to 0, status set to ACTIVE.")
		}

		fmt.Println("Budget reset execution complete.")
		return nil
	},
}

func init() {
	budgetResetCmd.Flags().String("campaign-id", "", "UUID of the campaign to reset")
	budgetResetCmd.Flags().Bool("reset-db-spend", false, "Reset current_spend to 0 and set status to ACTIVE in PostgreSQL")
	_ = budgetResetCmd.MarkFlagRequired("campaign-id")

	budgetCmd.AddCommand(budgetResetCmd)
	rootCmd.AddCommand(budgetCmd)
}
