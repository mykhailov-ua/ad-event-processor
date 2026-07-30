//go:build linux

package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func faultLabOnLinux() bool {
	return true
}

func stressNgAvailable() bool {
	_, err := exec.LookPath("stress-ng")
	return err == nil
}

func cpulimitAvailable() bool {
	_, err := exec.LookPath("cpulimit")
	return err == nil
}

func startStressNgHDD(t *testing.T, scratchFile string, sizeMB int, timeout time.Duration) (*exec.Cmd, error) {
	t.Helper()
	cmd := exec.Command(
		"stress-ng",
		"--hdd", "1",
		"--hdd-bytes", fmt.Sprintf("%dM", sizeMB),
		"--hdd-method", "read",
		"--timeout", strconv.Itoa(int(timeout.Seconds())),
		"--temp-path", filepath.Dir(scratchFile),
	)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	return cmd, nil
}

func throttleProcessCPU(t *testing.T, pid int, limitPct int) (*exec.Cmd, error) {
	t.Helper()
	cmd := exec.Command("cpulimit", "-p", strconv.Itoa(pid), "-l", strconv.Itoa(limitPct))
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
		}
	})
	return cmd, nil
}
