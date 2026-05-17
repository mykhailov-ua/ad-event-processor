package management

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NginxConfigWorker exports blacklist configuration files to a shared volume and signals Nginx to reload.
// Architectural Isolation (12-Factor App): Rather than executing OS binaries directly via exec syscalls from the HTTP gateway,
// this worker writes a flag file (reload_required.flg) to the shared directory.
// A separate lightweight process (e.g., inotify/cron) running inside the Nginx container must monitor this directory
// for the flag file, execute 'nginx -s reload', and remove the flag upon completion.
type NginxConfigWorker struct {
	svc        *Service
	exportPath string
}

func NewNginxConfigWorker(svc *Service, exportPath string) *NginxConfigWorker {
	return &NginxConfigWorker{
		svc:        svc,
		exportPath: exportPath,
	}
}

func (w *NginxConfigWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.ExportAndReload(ctx); err != nil {
				slog.Error("nginx export failed", "error", err)
			}
		}
	}
}

func (w *NginxConfigWorker) ExportAndReload(ctx context.Context) error {
	if len(w.svc.rdbs) == 0 || w.svc.rdbs[0] == nil {
		return fmt.Errorf("no redis client available")
	}
	rdb := w.svc.rdbs[0]

	manual, err := rdb.SMembers(ctx, "blacklist:manual").Result()
	if err != nil {
		return fmt.Errorf("failed to fetch manual blacklist: %w", err)
	}
	if err := w.writeDenyFile("manual.conf", manual); err != nil {
		return err
	}

	auto, err := rdb.SMembers(ctx, "blacklist:auto").Result()
	if err != nil {
		return fmt.Errorf("failed to fetch auto blacklist: %w", err)
	}
	if err := w.writeDenyFile("auto.conf", auto); err != nil {
		return err
	}

	flagPath := filepath.Join(w.exportPath, "reload_required.flg")
	if err := os.WriteFile(flagPath, []byte("1\n"), 0644); err != nil {
		return fmt.Errorf("failed to write reload flag: %w", err)
	}

	slog.Info("nginx blacklist exported and reload signaled via flag file", "manual_count", len(manual), "auto_count", len(auto))
	return nil
}

func (w *NginxConfigWorker) writeDenyFile(filename string, ips []string) error {
	if err := os.MkdirAll(w.exportPath, 0755); err != nil {
		return err
	}

	path := filepath.Join(w.exportPath, filename)
	var sb strings.Builder
	for _, ip := range ips {
		if ip == "" {
			continue
		}
		sb.WriteString("deny ")
		sb.WriteString(ip)
		sb.WriteString(";\n")
	}

	return os.WriteFile(path, []byte(sb.String()), 0644)
}
