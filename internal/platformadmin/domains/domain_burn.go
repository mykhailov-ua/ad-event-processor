package domains

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5"
)

type BurnDomainRequest struct {
	DeleteCloudflare bool `json:"delete_cloudflare"`
}

type BurnDomainResponse struct {
	Hostname          string `json:"hostname"`
	PoolStatus        string `json:"pool_status,omitempty"`
	CloudflareDeleted bool   `json:"cloudflare_deleted"`
}

func (dh *DomainHealth) BurnDomain(ctx context.Context, hostname string, req BurnDomainRequest) (BurnDomainResponse, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return BurnDomainResponse{}, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return BurnDomainResponse{}, fmt.Errorf("hostname is required")
	}

	var zoneID, recordID string
	err := dh.host.Pool().QueryRow(ctx, `
		SELECT COALESCE(cloudflare_zone_id, ''), COALESCE(dns_record_id, '')
		FROM domain_pool_domains
		WHERE hostname = $1`, host).Scan(&zoneID, &recordID)
	poolFound := !errors.Is(err, pgx.ErrNoRows)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return BurnDomainResponse{}, err
	}

	resp := BurnDomainResponse{Hostname: host}
	if poolFound {
		if err := dh.markPoolDomainBanned(ctx, host); err != nil {
			return BurnDomainResponse{}, err
		}
		resp.PoolStatus = "banned"
	}

	_, err = dh.host.Pool().Exec(ctx, `
		DELETE FROM domain_health_status
		WHERE hostname = $1 AND role = 'custom'`, host)
	if err != nil {
		return BurnDomainResponse{}, err
	}

	if req.DeleteCloudflare && zoneID != "" && recordID != "" {
		cf := dh.host.CloudflareClient()
		if cf != nil {
			if err := cf.DeleteDNSRecord(ctx, zoneID, recordID); err == nil {
				resp.CloudflareDeleted = true
				_, _ = dh.host.Pool().Exec(ctx, `
					UPDATE domain_pool_domains
					SET dns_record_id = NULL, updated_at = now()
					WHERE hostname = $1`, host)
			}
		}
	}

	if !poolFound && resp.PoolStatus == "" {
		return BurnDomainResponse{}, fmt.Errorf("domain not found in pool")
	}
	return resp, nil
}

func (dh *DomainHealth) poolDomainBanned(ctx context.Context, host string) (bool, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return false, fmt.Errorf("service unavailable")
	}
	var status string
	err := dh.host.Pool().QueryRow(ctx, `
		SELECT status FROM domain_pool_domains WHERE hostname = $1`, host).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return status == "banned", nil
}
