package runtimeautotune

import (
	"bufio"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"espx/internal/config"
)

// Apply sets process limits from host capacity when operators did not configure them.
func Apply(cfg *config.Config) {
	if _, ok := os.LookupEnv("GOMEMLIMIT"); !ok {
		if mem, err := systemMemoryBytes(); err == nil && mem > 0 {
			limit := int64(float64(mem) * 0.9)
			debug.SetMemoryLimit(limit)
		}
	}
	if cfg == nil {
		return
	}
	if _, ok := os.LookupEnv("MAX_WORKERS"); !ok {
		n := runtime.NumCPU()
		if n < 1 {
			n = 1
		}
		cfg.MaxWorkers = n
	}
}

func systemMemoryBytes() (uint64, error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "MemTotal:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			break
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, os.ErrNotExist
}
