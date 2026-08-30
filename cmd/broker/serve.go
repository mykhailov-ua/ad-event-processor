// Broker serve flags: resolves gnet addr, health addr, WAL paths, Redis coord URL.
//
// Defaults: gnet listen (9092 or runtimepaths.BrokerGnetSocket), health 8084, data-dir WAL root.
// WAL segment caps from -max-seg-mb and -index-kb passed to wireAndRunServe as MaxSegBytes / IndexInterval.
package main

import (
	"flag"
	"os"
	"strings"

	"ad-event-processor/pkg/netaddr"
	"ad-event-processor/pkg/runtimepaths"
)

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", "", "gnet listen address (tcp host:port or unix socket path)")
	healthAddr := fs.String("health-addr", "", "HTTP health/metrics address")
	dataDir := fs.String("data-dir", "/var/lib/ad-event-processor/broker", "WAL data directory")
	nodeID := fs.String("node-id", "broker-1", "Broker node ID")
	redisURL := fs.String("redis-url", "", "Redis URL or unix socket path for HA coordination")
	maxSegMB := fs.Int("max-seg-mb", 64, "Max segment size in MB")
	indexKB := fs.Int("index-kb", 4, "Index interval in KB")
	_ = fs.Parse(args)

	if *addr == "" {
		*addr = netaddr.ResolveListenAddr("127.0.0.1:9092", runtimepaths.BrokerGnetSocket())
	}
	if *healthAddr == "" {
		*healthAddr = netaddr.ResolveListenAddr("127.0.0.1:8084", runtimepaths.BrokerHealthSocket())
	}
	if *redisURL == "" {
		*redisURL = os.Getenv("BROKER_REDIS_URL")
	}
	if *redisURL == "" {
		if raw := os.Getenv("REDIS_ADDRS"); raw != "" {
			var first string
			if i := strings.Index(raw, ","); i >= 0 {
				first = strings.TrimSpace(raw[:i])
			} else {
				first = strings.TrimSpace(raw)
			}
			*redisURL = netaddr.RedisURLFromAddr(first, os.Getenv("REDIS_PASSWORD"), 0)
		}
	}

	wireAndRunServe(serveWireConfig{
		Addr:          *addr,
		HealthAddr:    *healthAddr,
		DataDir:       *dataDir,
		NodeID:        *nodeID,
		RedisURL:      *redisURL,
		MaxSegBytes:   int64(*maxSegMB) * 1024 * 1024,
		IndexInterval: int64(*indexKB) * 1024,
	})
}
