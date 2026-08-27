//go:build linux && license_guard

package licensing

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGuardNeedles_notPlaintextInSources(t *testing.T) {
	needles := guardSuspiciousMapNeedles()
	require.NotEmpty(t, needles)

	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(file)
	files := []string{
		filepath.Join(dir, "guard_needles_linux.go"),
		filepath.Join(dir, "guard_linux.go"),
		filepath.Join(dir, "guard_stretch_linux.go"),
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		require.NoError(t, err, path)
		for _, needle := range needles {
			require.False(t, containsBytes(raw, needle),
				"needle %q must not appear verbatim in %s", string(needle), filepath.Base(path))
		}
	}
}

func TestGuardNeedles_detectsFridaMapsLine(t *testing.T) {
	restore := SetGuardTracerPidReaderForTest(func() (int, error) { return 0, nil })
	t.Cleanup(restore)
	restoreMaps := SetGuardMapsScannerForTest(func() bool {
		line := []byte("7f0000000000-7f0000100000 r-xp 00000000 00:00 0 /tmp/frida-agent-64.so")
		lower := bytesToLower(line)
		for _, needle := range guardSuspiciousMapNeedles() {
			if containsBytes(lower, needle) {
				return true
			}
		}
		return false
	})
	t.Cleanup(restoreMaps)

	require.True(t, RunGuardProbeForTest())
	require.True(t, GuardTripped())
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		match := true
		for j := range needle {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func bytesToLower(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + ('a' - 'A')
		} else {
			out[i] = c
		}
	}
	return out
}
