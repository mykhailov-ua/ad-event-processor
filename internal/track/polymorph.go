package track

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	staticPolymorphWasmName  = "attest.wasm"
	staticPolymorphPixelName = "track_pixel.js"
	staticPolymorphManifest  = "polymorph.manifest"
)

// ApplyStaticPolymorphOverrides loads install-time variant static assets from dir.
// Expected layout: attest.wasm, track_pixel.js, optional polymorph.manifest (sha256 lines).
// Fail-open: missing dir or files keeps go:embed defaults.
func ApplyStaticPolymorphOverrides(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return nil
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("static polymorph: %q is not a directory", dir)
	}

	wasmPath := filepath.Join(dir, staticPolymorphWasmName)
	if wasmBytes, err := os.ReadFile(wasmPath); err == nil && len(wasmBytes) >= 8 {
		if wasmBytes[0] == 0 && wasmBytes[1] == 'a' && wasmBytes[2] == 's' && wasmBytes[3] == 'm' {
			AttestWasm = wasmBytes
			AttestWasmGnetResponse = buildTrackWasmGnetResponse(AttestWasm)
		}
	}

	pixelPath := filepath.Join(dir, staticPolymorphPixelName)
	if pixelBytes, err := os.ReadFile(pixelPath); err == nil && len(pixelBytes) > 0 {
		TrackPixelJS = pixelBytes
		TrackPixelGnetResponse = buildTrackClientJSGnetResponse(TrackPixelJS)
	}

	manifestPath := filepath.Join(dir, staticPolymorphManifest)
	if manifestBytes, err := os.ReadFile(manifestPath); err == nil {
		if err := verifyPolymorphManifest(string(manifestBytes), wasmPath, pixelPath); err != nil {
			return err
		}
	}
	return nil
}

func verifyPolymorphManifest(manifest, wasmPath, pixelPath string) error {
	expect := map[string]string{}
	for _, line := range strings.Split(manifest, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		expect[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}
	if want, ok := expect["attest_wasm_sha256"]; ok && want != "" {
		sum, err := fileSHA256Hex(wasmPath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(sum, want) {
			return fmt.Errorf("static polymorph: attest.wasm sha256 mismatch")
		}
	}
	if want, ok := expect["track_pixel_sha256"]; ok && want != "" {
		sum, err := fileSHA256Hex(pixelPath)
		if err != nil {
			return err
		}
		if !strings.EqualFold(sum, want) {
			return fmt.Errorf("static polymorph: track_pixel.js sha256 mismatch")
		}
	}
	return nil
}

func fileSHA256Hex(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func StaticPolymorphDirFromInstallRoot(installRoot string) string {
	root := strings.TrimSpace(installRoot)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "var", "static-polymorph")
}
