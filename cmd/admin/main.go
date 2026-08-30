// Admin CLI entrypoint. Package documentation: doc.go.
package main

import (
	"log/slog"
	"os"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := Execute(); err != nil {
		slog.Error("admin command failed", "error", err)
		os.Exit(1)
	}
}
