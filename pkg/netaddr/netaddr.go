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

// IsUnixSocketPath reports whether addr is a filesystem unix socket path.
func IsUnixSocketPath(addr string) bool {
	if addr == "" {
		return false
	}
	return strings.HasPrefix(addr, "/") || strings.HasSuffix(addr, ".sock") || strings.Contains(addr, ".sock")
}

// GnetListenURI returns a gnet listen URI (tcp:// or unix://).
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

// DialTimeout dials a TCP host:port or unix socket path.
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

// PrepareUnixSocket removes a stale socket file and ensures parent dir exists.
func PrepareUnixSocket(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir socket parent: %w", err)
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale socket: %w", err)
	}
	return nil
}

// ListenUnix prepares path and listens on unix.
func ListenUnix(path string) (net.Listener, error) {
	if err := PrepareUnixSocket(path); err != nil {
		return nil, err
	}
	return net.Listen("unix", path)
}

// ResolveListenAddr prefers unixPath when set, otherwise tcpAddr.
func ResolveListenAddr(tcpAddr, unixPath string) string {
	if strings.TrimSpace(unixPath) != "" {
		return unixPath
	}
	return tcpAddr
}

// RedisUniversalOptions builds go-redis options for TCP or unix socket addr.
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

// RedisClientOptions builds a single-shard client options for TCP or unix.
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

// ParseRedisURL opens redis from URL, unix:// path, or raw socket path.
func ParseRedisURL(raw string, password string) (redis.UniversalClient, error) {
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

// RedisURLFromAddr builds a redis URL for coordinator wiring.
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

// HTTPProbeTarget returns a URL suitable for GET health probes (http or unix).
func HTTPProbeTarget(tcpURL, unixSocket string) string {
	if strings.TrimSpace(unixSocket) != "" {
		return "http://unix/health#" + unixSocket
	}
	return tcpURL
}
