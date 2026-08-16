package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const defaultDomainPoolName = "default"

// SetCloudflareAPI overrides the Cloudflare client (tests).
func (s *Service) SetCloudflareAPI(api CloudflareAPI) {
	if s == nil {
		return
	}
	s.cloudflare = api
}

func (s *Service) cloudflareClient() CloudflareAPI {
	if s == nil {
		return nil
	}
	if s.cloudflare != nil {
		return s.cloudflare
	}
	if s.cfg == nil {
		return nil
	}
	return NewCloudflareClient(string(s.cfg.Management.CloudflareAPIToken), s.cfg.Management.CloudflareAPIBase)
}

func (s *Service) ParkDomain(ctx context.Context, req adminapi.ParkDomainRequest) (adminapi.ParkDomainResponse, error) {
	if s == nil || s.pool == nil {
		return adminapi.ParkDomainResponse{}, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(req.Domain)
	if host == "" {
		return adminapi.ParkDomainResponse{}, fmt.Errorf("domain is required")
	}
	zoneID := strings.TrimSpace(req.CloudflareZoneID)
	if zoneID == "" {
		return adminapi.ParkDomainResponse{}, fmt.Errorf("cloudflare_zone_id is required")
	}
	target := ""
	if s.cfg != nil {
		target = strings.TrimSpace(s.cfg.Management.CloudflareDNSTarget)
	}
	if target == "" {
		return adminapi.ParkDomainResponse{}, fmt.Errorf("cloudflare dns target not configured")
	}

	poolID, err := s.resolveDomainPoolID(ctx, req.PoolID)
	if err != nil {
		return adminapi.ParkDomainResponse{}, err
	}

	cf := s.cloudflareClient()
	if cf == nil {
		return adminapi.ParkDomainResponse{}, fmt.Errorf("cloudflare api not configured")
	}

	recordType := cloudflareRecordTypeForTarget(target)
	recordID, err := cf.CreateDNSRecord(ctx, zoneID, host, recordType, target, true)
	if err != nil {
		return adminapi.ParkDomainResponse{}, err
	}
	sslStatus, err := cf.ZoneSSLStatus(ctx, zoneID)
	if err != nil {
		sslStatus = "pending"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return adminapi.ParkDomainResponse{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var sortOrder int
	err = tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), -1) + 1
		FROM domain_pool_domains WHERE pool_id = $1`, poolID).Scan(&sortOrder)
	if err != nil {
		return adminapi.ParkDomainResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO domain_pool_domains (
			pool_id, hostname, sort_order, status, cloudflare_zone_id, dns_record_id, ssl_status
		) VALUES ($1, $2, $3, 'pending', $4, $5, $6)
		ON CONFLICT (pool_id, hostname) DO UPDATE SET
			cloudflare_zone_id = EXCLUDED.cloudflare_zone_id,
			dns_record_id = EXCLUDED.dns_record_id,
			ssl_status = EXCLUDED.ssl_status,
			updated_at = now()`,
		poolID, host, sortOrder, zoneID, recordID, sslStatus)
	if err != nil {
		return adminapi.ParkDomainResponse{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO domain_health_status (hostname, role)
		VALUES ($1, 'custom')
		ON CONFLICT (hostname) DO UPDATE SET updated_at = now()`, host)
	if err != nil {
		return adminapi.ParkDomainResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return adminapi.ParkDomainResponse{}, err
	}

	return adminapi.ParkDomainResponse{
		Success:     true,
		DNSRecordID: recordID,
		SSLStatus:   sslStatus,
		Hostname:    host,
		PoolID:      poolID,
	}, nil
}

func (s *Service) resolveDomainPoolID(ctx context.Context, poolID *uuid.UUID) (uuid.UUID, error) {
	if poolID != nil && *poolID != uuid.Nil {
		var one int
		err := s.pool.QueryRow(ctx, `SELECT 1 FROM domain_pools WHERE id = $1`, *poolID).Scan(&one)
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("domain pool not found")
		}
		if err != nil {
			return uuid.Nil, err
		}
		return *poolID, nil
	}
	var id uuid.UUID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO domain_pools (name) VALUES ($1)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id`, defaultDomainPoolName).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (s *Service) markPoolDomainBanned(ctx context.Context, hostname string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE domain_pool_domains
		SET status = 'banned', updated_at = now()
		WHERE hostname = $1 AND status <> 'banned'`, host)
	return err
}

func (s *Service) activatePoolDomain(ctx context.Context, hostname string) error {
	if s == nil || s.pool == nil {
		return nil
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE domain_pool_domains
		SET status = 'active', updated_at = now()
		WHERE hostname = $1 AND status = 'pending'`, host)
	return err
}
