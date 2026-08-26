package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"ad-event-processor/internal/openapi"
)

func main() {
	root, err := repoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := openapi.Export(root); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	slog.Info("openapi-export: wrote bundle",
		"routes", openapi.OpenAPIDir+"/"+openapi.GeneratedRoutesRel,
		"bundle", openapi.OpenAPIDir+"/"+openapi.BundleRel,
	)
}

func repoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("openapi-export: go.mod not found from %s", wd)
		}
		dir = parent
	}
}
