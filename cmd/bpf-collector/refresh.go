package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func (r *probeRun) refreshTargetsLoop(ctx context.Context) {
	ticker := time.NewTicker(r.refreshTargets)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.refreshTargetsFromScript(); err != nil {
				slog.Debug("refresh targets", "error", err)
			}
		}
	}
}

func (r *probeRun) refreshTargetsFromScript() error {
	targetsPath := filepath.Join(r.session.Dir, "targets.json")
	roles := r.rolesWanted
	if roles == "" {
		roles = "tracker,nginx,redis,processor"
	}
	script := filepath.Join(repoRoot(), "scripts", "test", "bpf_resolve_targets.sh")
	cmd := exec.CommandContext(context.Background(), "bash", script, targetsPath, roles)
	cmd.Env = os.Environ()
	cmd.Dir = repoRoot()
	if out, err := cmd.CombinedOutput(); err != nil {
		return err
	} else if len(out) > 0 {
		slog.Debug("refresh targets script", "output", string(out))
	}
	return r.mergeTargetsFile(targetsPath)
}

func (r *probeRun) mergeTargetsFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var meta sessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	if meta.RolesWanted != "" {
		r.rolesWanted = meta.RolesWanted
	}
	for _, t := range meta.Targets {
		if err := r.trackTarget(t.PID, t.CgroupID, t.Role, t.Name); err != nil {
			slog.Debug("refresh track target", "pid", t.PID, "error", err)
		}
	}
	return nil
}
