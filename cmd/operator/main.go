package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := Execute(); err != nil {
		slog.Error("operator command failed", "error", err)
		os.Exit(1)
	}
}
