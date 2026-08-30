//go:build linux && license_guard

package verify

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const GuardWatchdogCLIFlag = "--license-guard-watchdog"

const guardWatchdogReadyFD = 3

var (
	errPtraceBusy        = errors.New("ptrace busy")
	errPtraceUnavailable = errors.New("ptrace unavailable")
	guardPtraceLauncher  = launchPtraceWatchdog
)

func MaybeRunGuardWatchdogCLI(args []string) bool {
	if len(args) < 3 || args[1] != GuardWatchdogCLIFlag {
		return false
	}
	parentPID, err := strconv.Atoi(strings.TrimSpace(args[2]))
	if err != nil || parentPID <= 0 {
		os.Exit(1)
	}
	ready := os.NewFile(uintptr(guardWatchdogReadyFD), "guard-ready")
	runGuardWatchdogChild(parentPID, ready)
	os.Exit(0)
	return true
}

// launchPtraceWatchdog: child re-execs with PTRACE_ATTACH on parent; FD 3 signals ready to parent.
func launchPtraceWatchdog(ctx context.Context) {
	readyR, readyW, err := os.Pipe()
	if err != nil {
		slog.Warn("license guard: ptrace watchdog pipe", "error", err)
		return
	}

	exe, err := os.Executable()
	if err != nil {
		_ = readyR.Close()
		_ = readyW.Close()
		slog.Warn("license guard: ptrace watchdog executable", "error", err)
		return
	}

	cmd := exec.Command(exe, GuardWatchdogCLIFlag, strconv.Itoa(os.Getpid()))
	cmd.Env = os.Environ()
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.ExtraFiles = []*os.File{readyW}
	if err := cmd.Start(); err != nil {
		_ = readyR.Close()
		_ = readyW.Close()
		slog.Warn("license guard: ptrace watchdog start", "error", err)
		return
	}
	_ = readyW.Close()

	buf := make([]byte, 16)
	_ = readyR.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, readErr := readyR.Read(buf)
	_ = readyR.Close()
	processGuardWatchdogHandshake(string(buf[:n]), readErr)

	go func() {
		<-ctx.Done()
		if cmd.Process != nil {
			_ = cmd.Process.Signal(syscall.SIGTERM)
			done := make(chan struct{})
			go func() {
				_, _ = cmd.Process.Wait()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				_ = cmd.Process.Kill()
			}
		}
	}()
}

func processGuardWatchdogHandshake(msg string, readErr error) {
	switch strings.TrimSpace(msg) {
	case "ok":
		return
	case "busy":
		tripGuard("ptrace_busy")
	case "skip":
		if guardPtraceRequired {
			tripGuard("ptrace_skip")
			return
		}
		slog.Warn("license guard: ptrace watchdog unavailable (yama/ptrace_scope or permissions)")
	default:
		if readErr != nil {
			slog.Warn("license guard: ptrace watchdog handshake", "error", readErr)
		} else {
			slog.Warn("license guard: ptrace watchdog handshake failed", "reply", strings.TrimSpace(msg))
		}
	}
}

func runGuardWatchdogChild(parentPID int, ready *os.File) {
	defer func() {
		if ready != nil {
			_ = ready.Close()
		}
	}()

	if err := ptraceClaimParent(parentPID); err != nil {
		switch {
		case errors.Is(err, errPtraceBusy):
			_, _ = ready.Write([]byte("busy\n"))
			os.Exit(2)
		case errors.Is(err, errPtraceUnavailable):
			_, _ = ready.Write([]byte("skip\n"))
			os.Exit(0)
		default:
			_, _ = ready.Write([]byte("fail\n"))
			os.Exit(1)
		}
	}
	if _, err := ready.Write([]byte("ok\n")); err != nil {
		os.Exit(1)
	}
	monitorPtraceParent(parentPID)
}

func ptraceClaimParent(pid int) error {
	if err := unix.PtraceAttach(pid); err != nil {
		switch err {
		case unix.EBUSY:
			return errPtraceBusy
		case unix.EPERM, unix.EACCES:
			return errPtraceUnavailable
		default:
			return err
		}
	}

	var ws unix.WaitStatus
	for {
		wpid, err := unix.Wait4(pid, &ws, 0, nil)
		if err != nil {
			_ = unix.PtraceDetach(pid)
			return err
		}
		if wpid == pid {
			break
		}
	}
	if err := unix.PtraceCont(pid, 0); err != nil {
		_ = unix.PtraceDetach(pid)
		return err
	}
	return nil
}

func monitorPtraceParent(pid int) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		if err := unix.Kill(pid, 0); err != nil {
			os.Exit(0)
		}
	}
}

func SetGuardPtraceWatchdogLauncherForTest(fn func(context.Context)) func() {
	prev := guardPtraceLauncher
	if fn == nil {
		guardPtraceLauncher = launchPtraceWatchdog
	} else {
		guardPtraceLauncher = fn
	}
	return func() { guardPtraceLauncher = prev }
}

func ProcessGuardWatchdogHandshakeForTest(msg string, readErr error) {
	processGuardWatchdogHandshake(msg, readErr)
}

func SetGuardPtraceRequiredForTest(required bool) {
	guardPtraceRequired = required
}
