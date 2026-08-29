package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ad-event-processor/internal/broker"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/stream"
)

func runReplay(args []string) {
	fs := flag.NewFlagSet("replay", flag.ExitOnError)

	var (
		dataDir   = fs.String("data-dir", "var/lib/ad-event-processor/broker", "Path to broker WAL data directory")
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
		clickhouseConn, err := database.ConnectClickHouse(ctx, dsn)
		if err != nil {
			slog.Error("failed to connect to clickhouse", "error", err)
			os.Exit(1)
		}
		defer func() { _ = clickhouseConn.Close() }()

		st := stream.NewClickHouseStore(clickhouseConn, 30*time.Second, "", stream.ClickHouseSpoolConfig{}, nil)
		defer func() { _ = st.Close() }()
		store = st
	}

	replayer := broker.NewReplayer(broker.ReplayConfig{
		DataDir:       *dataDir,
		Topic:         *topic,
		From:          fromTime,
		To:            toTime,
		Target:        *target,
		ClickHouseDSN: dsn,
		BatchSize:     *batchSize,
	}, store, stream.ParseBrokerPayloadStream)

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
