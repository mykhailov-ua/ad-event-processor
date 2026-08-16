package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/bidshard/ad-event-processor/pkg/netaddr"
	"github.com/bidshard/ad-event-processor/pkg/runtimepaths"
)

func applyUnixTransportDefaults(cfg *Config) {
	cfg.TrackerUnixSocket = strings.TrimSpace(os.Getenv("TRACKER_UNIX_SOCKET"))
	cfg.ControlUnixSocket = strings.TrimSpace(os.Getenv("CONTROL_UNIX_SOCKET"))
	cfg.CHUnixSocket = strings.TrimSpace(os.Getenv("CH_UNIX_SOCKET"))
	if cfg.CHUnixSocket == "" {
		cfg.CHUnixSocket = runtimepaths.ClickHouseNativeSocket()
	}

	useUDS := getEnvBool("TRANSPORT_USE_UDS", applianceUDSDefault(cfg))

	if cfg.TrackerUnixSocket == "" && useUDS {
		inst := getEnvInt("TRACKER_INSTANCE", trackerInstanceFromPort(cfg.ServerPort))
		cfg.TrackerUnixSocket = runtimepaths.TrackerSocket(inst)
	}

	if strings.TrimSpace(os.Getenv("BROKER_URL")) == "" {
		if useUDS {
			cfg.Broker.URL = runtimepaths.BrokerGnetSocket()
		}
	}

	if strings.TrimSpace(os.Getenv("REGION_PROXY_ADDR")) == "" && useUDS {
		cfg.RegionProxyAddr = runtimepaths.RegionProxyGnetSocket()
	}

	if cfg.ControlUnixSocket == "" && useUDS {
		cfg.ControlUnixSocket = runtimepaths.ControlHTTPSocket()
	}

	if cfg.ControlUnixSocket != "" &&
		os.Getenv("CONTROL_URL") == "" &&
		os.Getenv("MANAGEMENT_URL") == "" {
		cfg.ManagementURL = "unix://" + cfg.ControlUnixSocket
	}

	if cfg.Broker.RedisURL == "" && len(cfg.RedisAddrs) > 0 {
		cfg.Broker.RedisURL = netaddr.RedisURLFromAddr(cfg.RedisAddrs[0], string(cfg.RedisPassword), 0)
	}
	if cfg.RegionProxyRedisURL == "" && len(cfg.RedisAddrs) > 0 {
		cfg.RegionProxyRedisURL = netaddr.RedisURLFromAddr(cfg.RedisAddrs[0], string(cfg.RedisPassword), 0)
	}
	if cfg.PgFailoverRedisURL == "" && len(cfg.RedisAddrs) > 0 {
		cfg.PgFailoverRedisURL = netaddr.RedisURLFromAddr(cfg.RedisAddrs[0], string(cfg.RedisPassword), 0)
	}

	applyCHUnixDSN(cfg)
}

func applianceUDSDefault(cfg *Config) bool {
	if len(cfg.RedisAddrs) == 0 {
		return false
	}
	return netaddr.IsUnixSocketPath(cfg.RedisAddrs[0])
}

func trackerInstanceFromPort(port string) int {
	p, err := strconv.Atoi(port)
	if err != nil || p < 8181 {
		return 0
	}
	return p - 8181
}

func applyCHUnixDSN(cfg *Config) {
	if !getEnvBool("CH_USE_UDS", applianceUDSDefault(cfg)) {
		return
	}
	unix := cfg.CHUnixSocket
	if unix == "" {
		return
	}
	dsn := string(cfg.CHDSN)
	if dsn != "" && !strings.Contains(dsn, "127.0.0.1") && !strings.Contains(dsn, "localhost") {
		return
	}
	user := envOrDefault("CH_USER", "default")
	pass := os.Getenv("CH_PASSWORD")
	db := envOrDefault("CH_NAME", "ad_event_processor")
	cfg.CHDSN = Secret(fmt.Sprintf(
		"clickhouse://%s:%s@/%s?protocol=native&secure=false&host=%s",
		user, pass, db, unix,
	))
	if string(cfg.CHReadonlyDSN) == "" || string(cfg.CHReadonlyDSN) == dsn {
		cfg.CHReadonlyDSN = cfg.CHDSN
	}
}
