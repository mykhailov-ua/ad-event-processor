//go:build linux && license_guard

package licensing

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	guardProcStatusPath = "/proc/self/status"
	guardProcMapsPath   = "/proc/self/maps"
	guardProcMemPath    = "/proc/self/mem"
)

var (
	guardTracerPidReader   = readTracerPID
	guardMapsScanner       = scanSuspiciousMaps
	guardTextHasher        = hashExecutableText
	guardTripRecorder      = func(string) {}
	guardTextBaseline      [32]byte
	guardTextBaselineValid bool
)

func GuardCompiledIn() bool { return true }

func StartLicenseGuard(ctx context.Context, cfg GuardConfig) {
	if !cfg.Enabled {
		return
	}
	if baseline, err := guardTextHasher(); err == nil {
		guardTextBaseline = baseline
		guardTextBaselineValid = true
	}
	if err := unix.Prctl(unix.PR_SET_DUMPABLE, 0, 0, 0, 0); err != nil {
		slog.Warn("license guard: prctl dumpable", "error", err)
	}
	if cfg.PtraceWatchdog {
		guardPtraceLauncher(ctx)
	}
	go guardLoop(ctx)
}

func guardLoop(ctx context.Context) {
	for {
		jitter := time.Duration(3+rand.IntN(5)) * time.Second
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitter):
			if runGuardProbe() {
				return
			}
		}
	}
}

func tripGuard(reason string) {
	if guardTripped.CompareAndSwap(0, 1) {
		InvalidateLicenseEpoch()
		recordGuardTrip(reason)
	}
}

func runGuardProbe() bool {
	if pid, err := guardTracerPidReader(); err == nil && pid > 0 {
		tripGuard("tracer_pid")
		return true
	}
	if guardMapsScanner() {
		tripGuard("suspicious_map")
		return true
	}
	if guardTextBaselineValid {
		if cur, err := guardTextHasher(); err == nil && !bytes.Equal(cur[:], guardTextBaseline[:]) {
			tripGuard("text_tamper")
			return true
		}
	}
	return GuardTripped()
}

func RunGuardProbeForTest() bool {
	return runGuardProbe()
}

func readTracerPID() (int, error) {
	f, err := os.Open(guardProcStatusPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "TracerPid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, errors.New("tracer pid field missing")
		}
		return strconv.Atoi(fields[1])
	}
	return 0, sc.Err()
}

func scanSuspiciousMaps() bool {
	data, err := os.ReadFile(guardProcMapsPath)
	if err != nil {
		return false
	}
	lower := bytes.ToLower(data)
	for _, needle := range guardSuspiciousMapNeedles() {
		if bytes.Contains(lower, needle) {
			return true
		}
	}
	return false
}

func guardSuspiciousMapNeedles() [][]byte {
	enc := [][]byte{
		{0x55, 0x41, 0x5a, 0x5a, 0x52},
		{0x54, 0x57, 0x56},
		{0x5f, 0x5f, 0x56, 0x57},
		{0x5a, 0x41, 0x5a, 0x5a, 0x52},
		{0x5a, 0x41, 0x5a, 0x5a, 0x5a, 0x5c, 0x5e},
	}
	out := make([][]byte, len(enc))
	for i, row := range enc {
		dec := make([]byte, len(row))
		for j, b := range row {
			dec[j] = b ^ 0x33
		}
		out[i] = dec
	}
	return out
}

func hashExecutableText() ([32]byte, error) {
	var zero [32]byte
	start, end, ok := executableTextRange()
	if !ok || end <= start {
		return zero, errors.New("text range unavailable")
	}
	f, err := os.Open(guardProcMemPath)
	if err != nil {
		return zero, err
	}
	defer f.Close()
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return zero, err
	}
	segment := make([]byte, end-start)
	if _, err := io.ReadFull(f, segment); err != nil {
		return zero, err
	}
	return sha256.Sum256(segment), nil
}

func executableTextRange() (int64, int64, bool) {
	data, err := os.ReadFile(guardProcMapsPath)
	if err != nil {
		return 0, 0, false
	}
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, " r-xp ") {
			continue
		}
		if exe != "" && !strings.Contains(line, exe) && !strings.Contains(line, "/memfd:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		parts := strings.Split(fields[0], "-")
		if len(parts) != 2 {
			continue
		}
		start, err1 := strconv.ParseInt(parts[0], 16, 64)
		end, err2 := strconv.ParseInt(parts[1], 16, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		return start, end, true
	}
	return 0, 0, false
}

func recordGuardTrip(reason string) {
	guardTripRecorder(reason)
}

func resetGuardHooksForTest() {
	guardTracerPidReader = readTracerPID
	guardMapsScanner = scanSuspiciousMaps
	guardTextHasher = hashExecutableText
	guardTripRecorder = func(string) {}
	guardPtraceLauncher = launchPtraceWatchdog
	guardTextBaselineValid = false
	guardTextBaseline = [32]byte{}
}

func SetGuardTracerPidReaderForTest(fn func() (int, error)) func() {
	prev := guardTracerPidReader
	if fn == nil {
		guardTracerPidReader = readTracerPID
	} else {
		guardTracerPidReader = fn
	}
	return func() { guardTracerPidReader = prev }
}

func SetGuardMapsScannerForTest(fn func() bool) func() {
	prev := guardMapsScanner
	if fn == nil {
		guardMapsScanner = scanSuspiciousMaps
	} else {
		guardMapsScanner = fn
	}
	return func() { guardMapsScanner = prev }
}

func SetGuardTextHasherForTest(fn func() ([32]byte, error)) func() {
	prev := guardTextHasher
	if fn == nil {
		guardTextHasher = hashExecutableText
	} else {
		guardTextHasher = fn
	}
	return func() { guardTextHasher = prev }
}

func SetGuardTextBaselineForTest(hash [32]byte) {
	guardTextBaseline = hash
	guardTextBaselineValid = true
}

func GuardTextFingerprint(hash [32]byte) uint32 {
	return binary.LittleEndian.Uint32(hash[:4])
}
