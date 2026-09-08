package wasmattest

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWasmAttest_polymorphVariants_holdout(t *testing.T) {
	if os.Getenv("WASM_POLYMORPH_GATE") != "1" {
		t.Skip("integration: run bash scripts/ci/static/wasm_polymorph_gate.sh")
	}
	raw := strings.TrimSpace(os.Getenv("WASM_POLYMORPH_PATHS"))
	if raw == "" {
		t.Fatal("WASM_POLYMORPH_PATHS required")
	}
	for _, path := range strings.Split(raw, ":") {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		data, err := os.ReadFile(path)
		require.NoError(t, err, path)
		require.NoError(t, VerifyModule(context.Background(), data, DefaultLimits()), path)
	}
}
