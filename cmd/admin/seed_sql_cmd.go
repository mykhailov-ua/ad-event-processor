package main

import (
	"os"

	"github.com/spf13/cobra"
)

var seedIngestSQLCmd = &cobra.Command{
	Use:   "seed-ingest-sql",
	Short: "Print ingest-only seed SQL with realistic deterministic UUIDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		writeSeedIngestSQL(os.Stdout, count)
		return nil
	},
}

var seedPrepTestSQLCmd = &cobra.Command{
	Use:   "seed-prep-test-sql",
	Short: "Print integration prep seed SQL for customers and campaigns",
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		writeSeedPrepTestSQL(os.Stdout, count)
		return nil
	},
}

var seedUUIDsShellCmd = &cobra.Command{
	Use:   "seed-uuids-shell",
	Short: "Print shell exports for deterministic seed UUIDs",
	RunE: func(cmd *cobra.Command, args []string) error {
		count, _ := cmd.Flags().GetInt("count")
		writeSeedUUIDShell(os.Stdout, count)
		return nil
	},
}

func init() {
	for _, command := range []*cobra.Command{seedIngestSQLCmd, seedPrepTestSQLCmd, seedUUIDsShellCmd} {
		command.Flags().Int("count", 100, "Number of seeded entities")
	}
	dbCmd.AddCommand(seedIngestSQLCmd)
	dbCmd.AddCommand(seedPrepTestSQLCmd)
	dbCmd.AddCommand(seedUUIDsShellCmd)
}
