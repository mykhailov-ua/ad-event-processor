package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"espx/internal/database"
	"espx/internal/domain"
	"espx/internal/ingestion"
	"espx/pkg/broker"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "replay":
		runReplay(os.Args[2:])
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: broker <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  replay    Replay historical WAL segments to target storage (e.g. clickhouse)")
	fmt.Println("\nOptions for replay:")
	fmt.Println("  --data-dir     Path to broker WAL data directory (e.g. /var/lib/bidshard/broker)")
	fmt.Println("  --topic        Topic name to replay (default: ad-events)")
	fmt.Println("  --from         RFC3339 start timestamp filter (e.g. 2026-08-08T12:00:00Z)")
	fmt.Println("  --to           RFC3339 end timestamp filter (e.g. 2026-08-08T18:00:00Z)")
	fmt.Println("  --target       Target system: clickhouse, stdout, or null (default: clickhouse)")
	fmt.Println("  --ch-dsn       ClickHouse DSN connection string")
	fmt.Println("  --batch-size   Replay batch size (default: 50000)")
}

func runReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)

	var (
		dataDir   = fs.String("data-dir", "var/lib/bidshard/broker", "Path to broker WAL data directory")
		topic     = fs.String("topic", "ad-events", "Topic name")
		fromStr   = fs.String("from", "", "RFC3339 start timestamp filter")
		toStr     = fs.String("to", "", "RFC3339 end timestamp filter")
		target    = fs.String("target", "clickhouse", "Target system (clickhouse, stdout, null)")
		chDSN     = fs.String("ch-dsn", "", "ClickHouse DSN (or CH_DSN env)")
		batchSize = fs.Int("batch-size", 50000, "Replay batch size")
	)

	_ = fs.Parse(args)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	var fromTime, toTime time.Time
	var err error

	if *fromStr != "" {
		fromTime, err = time.Parse(time.RFC3339, *fromStr)
		if err != nil {
			slog.Error("invalid --from timestamp format (must be RFC3339)", "error", err)
			os.Exit(1)
		}
	}
	if *toStr != "" {
		toTime, err = time.Parse(time.RFC3339, *toStr)
		if err != nil {
			slog.Error("invalid --to timestamp format (must be RFC3339)", "error", err)
			os.Exit(1)
		}
	}

	dsn := *chDSN
	if dsn == "" {
		dsn = os.Getenv("CH_DSN")
	}

	var store domain.EventStore

	if *target == "clickhouse" {
		if dsn == "" {
			slog.Error("clickhouse DSN is required via --ch-dsn or CH_DSN env")
			os.Exit(1)
		}
		chConn, err := database.ConnectClickHouse(ctx, dsn)
		if err != nil {
			slog.Error("failed to connect to clickhouse", "error", err)
			os.Exit(1)
		}
		defer func() { _ = chConn.Close() }()

		chStore := ingestion.NewClickHouseStore(chConn, 30*time.Second, "", ingestion.CHSpoolConfig{}, nil)
		defer func() { _ = chStore.Close() }()
		store = chStore
	}

	replayer := broker.NewReplayer(broker.ReplayConfig{
		DataDir:   *dataDir,
		Topic:     *topic,
		From:      fromTime,
		To:        toTime,
		Target:    *target,
		CHDSN:     dsn,
		BatchSize: *batchSize,
	}, store)

	slog.Info("starting broker replay",
		"data_dir", *dataDir,
		"topic", *topic,
		"from", *fromStr,
		"to", *toStr,
		"target", *target,
	)

	start := time.Now()
	res, err := replayer.Replay(ctx)
	if err != nil {
		slog.Error("broker replay failed", "error", err)
		os.Exit(1)
	}

	slog.Info("broker replay completed successfully",
		"events_read", res.EventsRead,
		"events_replayed", res.EventsReplayed,
		"sha256_hash", res.PayloadHash,
		"duration", time.Since(start).String(),
	)
}
