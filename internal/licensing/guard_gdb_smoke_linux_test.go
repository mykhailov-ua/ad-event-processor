//go:build linux && license_guard

package licensing_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/stretchr/testify/require"
)

const guardGDBChildEnv = "LICENSE_GUARD_GDB_CHILD"

func TestGuard_GDBAttachDenied(t *testing.T) {
	if os.Getenv("LICENSE_GDB_SMOKE") != "1" {
		t.Skip("set LICENSE_GDB_SMOKE=1 for gdb attach lab smoke")
	}
	if _, err := exec.LookPath("gdb"); err != nil {
		t.Skip("gdb not in PATH")
	}

	exe, err := os.Executable()
	require.NoError(t, err)

	child := exec.Command(exe, "-test.run=^TestGuard_GDBAttachDeniedChild$", "-test.count=1", "-test.timeout=60s")
	child.Env = append(os.Environ(), guardGDBChildEnv+"=1")
	child.Stdout = nil
	child.Stderr = nil
	require.NoError(t, child.Start())
	t.Cleanup(func() {
		if child.Process != nil {
			_ = child.Process.Signal(syscall.SIGTERM)
		}
	})
	time.Sleep(500 * time.Millisecond)

	gdb := exec.Command(
		"gdb", "-batch",
		"-ex", "set pagination off",
		"-ex", "attach "+strconv.Itoa(child.Process.Pid),
		"-ex", "detach",
		"-ex", "quit",
	)
	var gdbOut bytes.Buffer
	gdb.Stdout = &gdbOut
	gdb.Stderr = &gdbOut
	_ = gdb.Run()

	out := gdbOut.String()
	t.Logf("fault_proof fault=license_guard_gdb_attach harness=license_guard_release gdb_out=%q", out)

	denied := strings.Contains(out, "Operation not permitted") ||
		strings.Contains(out, "Tracing is already") ||
		strings.Contains(out, "Already tracing") ||
		strings.Contains(out, "EBUSY") ||
		strings.Contains(out, "ptrace:")
	require.True(t, denied, "expected gdb attach denial, got: %s", out)
}

func TestGuard_GDBAttachDeniedChild(t *testing.T) {
	if os.Getenv(guardGDBChildEnv) != "1" {
		t.Skip("child harness only")
	}

	licensing.ResetGuardForTest()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	licensing.StartLicenseGuard(ctx, licensing.GuardConfig{
		Enabled:        true,
		PtraceWatchdog: true,
	})

	select {
	case <-ctx.Done():
	case <-time.After(45 * time.Second):
	}
}
