package fraud

import (
	"context"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const residentialIntelCHInsert = `
INSERT INTO ad_event_processor.residential_intel_cache (
 ip_hash, residential_proxy, vpn, proxy, provider, cached_at
) VALUES (?, ?, ?, ?, ?, ?)`

func insertResidentialIntelCH(ctx context.Context, conn driver.Conn, ip string, result ResidentialIntelResult, provider string, cachedAt time.Time) error {
	if conn == nil {
		return nil
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ErrInvalidIP
	}
	hasher := clickhousePIIHasher()
	if hasher == nil {
		return fmt.Errorf("residential intel ch: pii hasher unavailable")
	}
	var residential, vpn, proxy uint8
	if result.ResidentialProxy {
		residential = 1
	}
	if result.VPN {
		vpn = 1
	}
	if result.Proxy {
		proxy = 1
	}
	if err := conn.Exec(ctx, residentialIntelCHInsert,
		piihash.FixedString16(hasher.HashIP(ip)),
		residential,
		vpn,
		proxy,
		provider,
		cachedAt,
	); err != nil {
		return fmt.Errorf("insert residential_intel_cache: %w", err)
	}
	return nil
}
