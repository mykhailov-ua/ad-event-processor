package fraud

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/fraud/features"
)

func testModelPath(t *testing.T) string {
	t.Helper()
	if path := os.Getenv("FRAUD_TEST_MODEL"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		t.Skipf("FRAUD_TEST_MODEL not found: %s", path)
	}
	candidates := []string{
		filepath.Join("..", "..", "var", "fraudscore", "artifacts", "model.txt"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	t.Skip("fraud model not found; run make fraud-modeling-check locally")
	return ""
}

func ipHashHex(ip string) string {
	h := features.HashIPForClickhouse(ip)
	return hex.EncodeToString(h[:])
}
