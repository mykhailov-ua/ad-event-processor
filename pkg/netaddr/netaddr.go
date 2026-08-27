package netaddr

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

func IsUnixSocketPath(addr string) bool {
	if addr == "" {
		return false
	}
	return strings.HasPrefix(addr, "/") || strings.HasSuffix(addr, ".sock") || strings.Contains(addr, ".sock")
}

func GnetListenURI(addr string) string {
	if strings.HasPrefix(addr, "tcp://") || strings.HasPrefix(addr, "unix://") {
		return addr
	}
	if IsUnixSocketPath(addr) {
		return "unix://" + addr
	}
	if strings.Contains(addr, ":") {
		return "tcp://" + addr
	}
	return "tcp://" + addr
}

func DialTimeout(addr string, timeout time.Duration) (net.Conn, error) {
	if IsUnixSocketPath(addr) {
		var d net.Dialer
		if timeout > 0 {
			d.Timeout = timeout
		}
		return d.Dial("unix", addr)
	}
	return net.DialTimeout("tcp", addr, timeout)
}

func PrepareUnixSocket(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir socket parent: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}

func EnsureUnixSocketWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("not a unix socket: %s", path)
	}
	if info.Mode().Perm()&0o222 == 0o222 {
		return nil
	}
	return os.Chmod(path, info.Mode().Perm()|0o222)
}

func ListenUnix(path string) (net.Listener, error) {
	if err := PrepareUnixSocket(path); err != nil {
		return nil, err
	}
	return net.Listen("unix", path)
}

func ResolveListenAddr(tcpAddr, unixPath string) string {
	if strings.TrimSpace(unixPath) != "" {
		return unixPath
	}
	return tcpAddr
}

func RedisUniversalOptions(addr, password string) *redis.UniversalOptions {
	uopts := &redis.UniversalOptions{
		Addrs:    []string{addr},
		Password: password,
	}
	if IsUnixSocketPath(addr) {
		uopts.Dialer = func(ctx context.Context, _, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", addr)
		}
	}
	return uopts
}

func RedisClientOptions(addr, password string) *redis.Options {
	if IsUnixSocketPath(addr) {
		return &redis.Options{
			Network:  "unix",
			Addr:     addr,
			Password: password,
		}
	}
	return &redis.Options{
		Addr:     addr,
		Password: password,
	}
}

func ParseRedisURL(raw, password string) (redis.UniversalClient, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("redis url is empty")
	}
	if strings.HasPrefix(raw, "unix://") {
		path := strings.TrimPrefix(raw, "unix://")
		if i := strings.Index(path, "?"); i >= 0 {
			path = path[:i]
		}
		return redis.NewClient(RedisClientOptions(path, password)), nil
	}
	if IsUnixSocketPath(raw) {
		return redis.NewClient(RedisClientOptions(raw, password)), nil
	}
	opts, err := redis.ParseURL(raw)
	if err != nil {
		return nil, err
	}
	if password != "" {
		opts.Password = password
	}
	return redis.NewClient(opts), nil
}

func RedisURLFromAddr(addr, password string, db int) string {
	if IsUnixSocketPath(addr) {
		if password != "" {
			return fmt.Sprintf("unix://%s?password=%s&db=%d", addr, password, db)
		}
		return fmt.Sprintf("unix://%s?db=%d", addr, db)
	}
	if password != "" {
		return fmt.Sprintf("redis://:%s@%s/%d", password, addr, db)
	}
	return fmt.Sprintf("redis://%s/%d", addr, db)
}

func HTTPProbeTarget(tcpURL, unixSocket string) string {
	if strings.TrimSpace(unixSocket) != "" {
		return "http://unix/health#" + unixSocket
	}
	return tcpURL
}
