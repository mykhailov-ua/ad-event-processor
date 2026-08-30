// Target PID and cgroup registration in BPF target_pids / target_cgroups maps.
//
// Role:
//   - trackTarget writes role byte to BPF maps and updates in-memory tracked map for dump labels.
//   - trackPID is cgroup_id=0 shorthand; duplicate PID updates cgroup only when cgroup_id changes.
//
// Invariants:
//   - pid==0 && cgroup_id==0 returns error; pid==0 with cgroup tracks cgroup-only (redis containers).
//   - roleName maps role constants to load-report role strings.
//
// Verify:
//
//	grep tracking pid logs during bpf-collector run
package main

import (
	"fmt"
	"log/slog"
)

func (r *probeRun) trackPID(pid uint32, role uint8, name string) error {
	return r.trackTarget(pid, 0, role, name)
}

func (r *probeRun) trackTarget(pid uint32, cgroupID uint64, role uint8, name string) error {
	if pid == 0 && cgroupID == 0 {
		return fmt.Errorf("invalid target")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if pid != 0 {
		if existing, ok := r.tracked[pid]; ok {
			if cgroupID != 0 && cgroupID != existing.CgroupID {
				_ = r.coll.PutTargetCgroup(cgroupID, role)
			}
			return nil
		}
		if err := r.coll.PutTargetPID(pid, role); err != nil {
			return err
		}
	}
	if cgroupID != 0 {
		if err := r.coll.PutTargetCgroup(cgroupID, role); err != nil {
			return err
		}
	}
	if pid != 0 {
		r.tracked[pid] = targetEntry{PID: pid, CgroupID: cgroupID, Role: role, Name: name}
		slog.Info("tracking pid", "pid", pid, "cgroup_id", cgroupID, "role", roleName(role), "name", name)
	} else if cgroupID != 0 {
		slog.Info("tracking cgroup", "cgroup_id", cgroupID, "role", roleName(role), "name", name)
	}
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
	case roleLoadgen:
		return "loadgen"
	case roleProcessor:
		return "processor"
	default:
		return "unknown"
	}
}
