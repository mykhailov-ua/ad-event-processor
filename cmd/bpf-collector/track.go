package main

import (
	"fmt"
	"log/slog"
)

func (r *probeRun) trackPID(pid uint32, role uint8, name string) error {
	if pid == 0 {
		return fmt.Errorf("invalid pid")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tracked[pid]; ok {
		return nil
	}
	if err := r.coll.PutTargetPID(pid, role); err != nil {
		return err
	}
	r.tracked[pid] = targetEntry{PID: pid, Role: role, Name: name}
	slog.Info("tracking pid", "pid", pid, "role", roleName(role), "name", name)
	return nil
}

func roleName(role uint8) string {
	switch role {
	case roleTracker:
		return "tracker"
	case roleNginx:
		return "nginx"
	case roleRedis:
		return "redis"
	case roleK6:
		return "k6"
	case roleProcessor:
		return "processor"
	default:
		return "unknown"
	}
}
