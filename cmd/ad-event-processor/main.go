package main

import (
	"log/slog"
	"os"

	"github.com/bidshard/ad-event-processor/cmd/ad-event-processor/cmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := cmd.Execute(); err != nil {
		slog.Error("ad-event-processor command failed", "error", err)
		os.Exit(1)
	}
}
