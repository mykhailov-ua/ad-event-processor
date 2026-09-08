package track

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestApplyStaticPolymorphOverrides_holdout(t *testing.T) {
	dir := t.TempDir()
	wasm := []byte{0, 'a', 's', 'm', 1, 0, 0, 0, 0}
	pixel := []byte("/*aed:test*/function trackEvent(){}")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "attest.wasm"), wasm, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "track_pixel.js"), pixel, 0o644))

	beforeWasm := append([]byte(nil), AttestWasm...)
	beforePixel := append([]byte(nil), TrackPixelJS...)
	t.Cleanup(func() {
		AttestWasm = beforeWasm
		TrackPixelJS = beforePixel
		AttestWasmGnetResponse = buildTrackWasmGnetResponse(AttestWasm)
		TrackPixelGnetResponse = buildTrackClientJSGnetResponse(TrackPixelJS)
	})

	require.NoError(t, ApplyStaticPolymorphOverrides(dir))
	require.Equal(t, wasm, AttestWasm)
	require.Equal(t, pixel, TrackPixelJS)
	require.NotEqual(t, beforeWasm, AttestWasm)
	require.NotEqual(t, beforePixel, TrackPixelJS)

	resp, ok := TrackClientStaticGnetResponse([]byte(AttestWasmPath))
	require.True(t, ok)
	require.Contains(t, string(resp), "application/wasm")
}
