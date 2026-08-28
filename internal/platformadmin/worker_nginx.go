package platformadmin

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/redis/go-redis/v9"
)

type NginxHost interface {
	ControlRedis() redis.UniversalClient
}

type NginxWorker struct {
	host       NginxHost
	exportPath string
}

func NewNginxWorker(host NginxHost, exportPath string) *NginxWorker {
	return &NginxWorker{
		host:       host,
		exportPath: exportPath,
	}
}

func NewNginxConfigWorker(host NginxHost, exportPath string) *NginxWorker {
	return NewNginxWorker(host, exportPath)
}

func (nc *NginxWorker) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := nc.ExportAndReload(ctx); err != nil {
				slog.Error("nginx export failed", "err", err)
			}
		}
	}
}

func (nc *NginxWorker) ExportAndReload(ctx context.Context) error {
	redisClient := nc.host.ControlRedis()
	if redisClient == nil {
		return fmt.Errorf("no healthy redis shard")
	}

	manual, err := redisClient.SMembers(ctx, "blacklist:manual").Result()
	if err != nil {
		return fmt.Errorf("failed to fetch manual blacklist: %w", err)
	}
	if err := nc.writeDenyFile("manual.conf", manual); err != nil {
		return err
	}

	auto, err := redisClient.SMembers(ctx, "blacklist:auto").Result()
	if err != nil {
		return fmt.Errorf("failed to fetch auto blacklist: %w", err)
	}
	if err := nc.writeDenyFile("auto.conf", auto); err != nil {
		return err
	}

	flagPath := filepath.Join(nc.exportPath, "reload_required.flg")
	if err := os.WriteFile(flagPath, []byte("1\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write reload flag: %w", err)
	}

	slog.Info("nginx blacklist exported and reload signaled via flag file", "manual_count", len(manual), "auto_count", len(auto))
	return nil
}

func (nc *NginxWorker) writeDenyFile(filename string, ips []string) (err error) {
	if err := os.MkdirAll(nc.exportPath, 0o755); err != nil {
		return err
	}

	path := filepath.Join(nc.exportPath, filename)
	tmpPath := path + ".tmp"

	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open temp config file: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tmpFile.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	bw := bufio.NewWriter(tmpFile)
	for _, ip := range ips {
		if ip == "" {
			continue
		}

		if net.ParseIP(ip) == nil {
			if _, _, errCIDR := net.ParseCIDR(ip); errCIDR != nil {
				slog.Warn("skipping invalid blacklist IP/CIDR to prevent injection", "ip", ip)
				continue
			}
		}

		if _, err = bw.WriteString("deny "); err != nil {
			return fmt.Errorf("failed to write directive prefix: %w", err)
		}
		if _, err = bw.WriteString(ip); err != nil {
			return fmt.Errorf("failed to write IP: %w", err)
		}
		if _, err = bw.WriteString(";\n"); err != nil {
			return fmt.Errorf("failed to write directive suffix: %w", err)
		}
	}

	if err = bw.Flush(); err != nil {
		return fmt.Errorf("failed to flush config buffer: %w", err)
	}

	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync config file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp config file: %w", err)
	}

	if err = os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to atomically replace config file: %w", err)
	}

	return nil
}

func (nc *NginxWorker) WriteDenyFile(filename string, ips []string) error {
	return nc.writeDenyFile(filename, ips)
}

func (nc *NginxWorker) ExportPath() string {
	return nc.exportPath
}

func NewNginxWorkerForTest(exportPath string) *NginxWorker {
	return &NginxWorker{exportPath: exportPath}
}
