package wasmattest

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/pkg/antifraudtelemetry"

	"github.com/stretchr/testify/require"
)

func loadTestModule(t *testing.T) []byte {
	t.Helper()
	root := findModuleRoot(t)
	path := filepath.Join(root, "var", "wasm", "attest.wasm")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skip("integration: run bash scripts/build/wasm_attest.sh (wasm module missing)")
	}
	return data
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	dir := wd
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("go.mod not found")
	return ""
}

func TestWasmAttest_verifyModule_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	require.NoError(t, VerifyModule(wasm, DefaultLimits()))
}

func TestWasmAttest_powParity_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	ctx := t.Context()
	sb, err := LoadSandbox(ctx, wasm, DefaultLimits())
	require.NoError(t, err)
	defer sb.Close(ctx)

	require.Equal(t, uint32(ABIVersion), sb.ABIVersion())

	salt := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	difficulty := uint8(2)
	nonce, err := sb.PoWSolve(ctx, salt, difficulty, 500_000)
	require.NoError(t, err)
	require.True(t, antifraudtelemetry.VerifyPoW(salt, nonce, difficulty))
}

func TestWasmAttest_floatNoiseIEEE_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	ctx := t.Context()
	sb, err := LoadSandbox(ctx, wasm, DefaultLimits())
	require.NoError(t, err)
	defer sb.Close(ctx)

	got, err := sb.FloatNoiseIEEE(ctx)
	require.NoError(t, err)
	require.Equal(t, FloatNoiseIEEEHex, hex.EncodeToString(got))
}

func TestWasmAttest_benchDeterministic_holdout(t *testing.T) {
	wasm := loadTestModule(t)
	ctx := t.Context()
	sb, err := LoadSandbox(ctx, wasm, DefaultLimits())
	require.NoError(t, err)
	defer sb.Close(ctx)

	a, err := sb.BenchMul(ctx, 100_000)
	require.NoError(t, err)
	b, err := sb.BenchMul(ctx, 100_000)
	require.NoError(t, err)
	require.Equal(t, a, b)
}
