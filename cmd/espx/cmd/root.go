package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"espx/internal/config"

	"github.com/spf13/cobra"
)

var (
	envPath string
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "espx",
	Short: "eSPX operator CLI",
	Long:  "Operator-facing CLI for health checks, MVSS checklist, and support bundles.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return loadEnvFile(envPath)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func loadEnvFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
			(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
			val = val[1 : len(val)-1]
		}
		if os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}
	return scanner.Err()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&envPath, "env-path", ".env", "path to .env configuration file")
}

func ensureConfig(only []string) error {
	if cfg != nil {
		return nil
	}
	if !needsFullConfig(only) {
		return nil
	}
	c, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg = c
	return nil
}

func needsFullConfig(only []string) bool {
	if len(only) == 0 {
		return true
	}
	for _, name := range only {
		switch strings.ToLower(strings.TrimSpace(name)) {
		case "redis", "clickhouse", "tls":
			return true
		}
	}
	return false
}
