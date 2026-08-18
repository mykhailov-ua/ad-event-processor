package cpuset

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

func Count(list string) int {
	list = strings.TrimSpace(list)
	if list == "" {
		return 0
	}
	seen := make(map[int]struct{})
	for part := range strings.SplitSeq(list, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.Index(part, "-"); i >= 0 {
			lo, err1 := strconv.Atoi(strings.TrimSpace(part[:i]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(part[i+1:]))
			if err1 != nil || err2 != nil || hi < lo {
				continue
			}
			for cpu := lo; cpu <= hi; cpu++ {
				seen[cpu] = struct{}{}
			}
			continue
		}
		cpu, err := strconv.Atoi(part)
		if err != nil {
			continue
		}
		seen[cpu] = struct{}{}
	}
	return len(seen)
}

func EffectiveCount() (int, error) {
	paths := []string{
		"/sys/fs/cgroup/cpuset.cpus.effective",
		"/sys/fs/cgroup/cpuset.cpus",
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		n := Count(strings.TrimSpace(string(data)))
		if n > 0 {
			return n, nil
		}
	}
	return fromProcStatus()
}

func fromProcStatus() (int, error) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "Cpus_allowed_list:") {
			continue
		}
		list := strings.TrimSpace(strings.TrimPrefix(line, "Cpus_allowed_list:"))
		n := Count(list)
		if n > 0 {
			return n, nil
		}
		break
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, os.ErrNotExist
}
