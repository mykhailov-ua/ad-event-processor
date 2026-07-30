package main

import (
	"log/slog"
	"os"

	"espx/cmd/espx/cmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := cmd.Execute(); err != nil {
		slog.Error("espx command failed", "error", err)
		os.Exit(1)
	}
}
