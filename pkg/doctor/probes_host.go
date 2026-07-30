package doctor

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

type KernelProbe struct{}

func (KernelProbe) Name() string { return "kernel" }

func (KernelProbe) Run(ctx context.Context) Result {
	start := time.Now()
	if runtime.GOOS != "linux" {
		return Result{Name: "kernel", Status: StatusSkip, Detail: "linux only", Latency: time.Since(start).Milliseconds()}
	}
	if hasConfigXDP() {
		return Result{Name: "kernel", Status: StatusPass, Detail: "CONFIG_XDP enabled", Latency: time.Since(start).Milliseconds()}
	}
	_, _, errno := unix.Syscall(unix.SYS_BPF, 0, 0, 0)
	if errno == unix.ENOSYS {
		return Result{Name: "kernel", Status: StatusFail, Detail: "bpf syscall unavailable", Latency: time.Since(start).Milliseconds()}
	}
	return Result{Name: "kernel", Status: StatusPass, Detail: "bpf syscall available", Latency: time.Since(start).Milliseconds()}
}

func hasConfigXDP() bool {
	paths := []string{
		"/proc/config.gz",
		"/boot/config-" + runtime.GOARCH,
	}
	if kernelRelease, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		paths = append(paths, "/boot/config-"+strings.TrimSpace(string(kernelRelease)))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(data)
		if strings.Contains(text, "CONFIG_XDP=y") || strings.Contains(text, "CONFIG_XDP=m") {
			return true
		}
	}
	return false
}

type SysctlProbe struct{}

func (SysctlProbe) Name() string { return "sysctl" }

func (SysctlProbe) Run(ctx context.Context) Result {
	start := time.Now()
	fileMax, err := readSysctlInt("/proc/sys/fs/file-max")
	if err != nil {
		return Result{Name: "sysctl", Status: StatusFail, Detail: err.Error(), Latency: time.Since(start).Milliseconds()}
	}
	somaxconn, err := readSysctlInt("/proc/sys/net/core/somaxconn")
	if err != nil {
		return Result{Name: "sysctl", Status: StatusFail, Detail: err.Error(), Latency: time.Since(start).Milliseconds()}
	}
	if fileMax < 1_000_000 {
		return Result{
			Name: "sysctl", Status: StatusFail,
			Detail:  fmt.Sprintf("fs.file-max=%d want >= 1000000", fileMax),
			Latency: time.Since(start).Milliseconds(),
		}
	}
	if somaxconn < 4096 {
		return Result{
			Name: "sysctl", Status: StatusFail,
			Detail:  fmt.Sprintf("net.core.somaxconn=%d want >= 4096", somaxconn),
			Latency: time.Since(start).Milliseconds(),
		}
	}
	return Result{
		Name: "sysctl", Status: StatusPass,
		Detail:  fmt.Sprintf("file-max=%d somaxconn=%d", fileMax, somaxconn),
		Latency: time.Since(start).Milliseconds(),
	}
}

func readSysctlInt(path string) (int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	val, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", path, err)
	}
	return val, nil
}
