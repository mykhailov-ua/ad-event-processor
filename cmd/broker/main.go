package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/pkg/broker"
	"github.com/bidshard/ad-event-processor/pkg/lifecycle"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "--health-probe" {
		if !lifecycle.RunHealthProbe(os.Args[2]) {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if len(os.Args) >= 2 && (os.Args[1] == "serve" || strings.HasPrefix(os.Args[1], "-")) {
		if os.Args[1] == "serve" {
			runServe(os.Args[2:])
			return
		}
		runServe(os.Args[1:])
		return
	}

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
	_, _ = fmt.Fprintf(os.Stdout, "Usage: broker <command> [options]\n")
	_, _ = fmt.Fprintf(os.Stdout, "\nCommands:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  serve      Run mmap WAL broker (gnet ingress)\n")
	_, _ = fmt.Fprintf(os.Stdout, "  replay     Replay historical WAL segments to target storage (e.g. clickhouse)\n")
	_, _ = fmt.Fprintf(os.Stdout, "\nOptions for replay:\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --data-dir     Path to broker WAL data directory (e.g. /var/lib/bidshard/broker)\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --topic        Topic name to replay (default: ad-events)\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --from         RFC3339 start timestamp filter (e.g. 2026-08-08T12:00:00Z)\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --to           RFC3339 end timestamp filter (e.g. 2026-08-08T18:00:00Z)\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --target       Target system: clickhouse, stdout, or null (default: clickhouse)\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --ch-dsn       ClickHouse DSN connection string\n")
	_, _ = fmt.Fprintf(os.Stdout, "  --batch-size   Replay batch size (default: 50000)\n")
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
