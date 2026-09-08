package domains

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type WildcardSSLRequest struct {
	CloudflareZoneID string     `json:"cloudflare_zone_id"`
	ZoneName         string     `json:"zone_name"`
	PoolID           *uuid.UUID `json:"pool_id,omitempty"`
	IncludeApex      bool       `json:"include_apex"`
}

type WildcardSSLResponse struct {
	ID                string     `json:"id"`
	PoolID            string     `json:"pool_id"`
	WildcardHostname  string     `json:"wildcard_hostname"`
	AcmeState         string     `json:"acme_state"`
	SSLNotAfter       *time.Time `json:"ssl_not_after,omitempty"`
	CloudflareProxied bool       `json:"cloudflare_proxied"`
	Message           string     `json:"message,omitempty"`
}

var defaultWildcardIssuer wildcardCertIssuer = &acmeDNS01Issuer{}

func (dh *DomainHealth) ListCloudflareZones(ctx context.Context) ([]CloudflareZone, error) {
	if dh == nil || dh.host == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	cf := dh.cloudflareAPI()
	if cf == nil {
		return nil, fmt.Errorf("cloudflare api not configured")
	}
	return cf.ListZones(ctx)
}

func (dh *DomainHealth) SetupWildcardSSL(ctx context.Context, req WildcardSSLRequest) (WildcardSSLResponse, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return WildcardSSLResponse{}, fmt.Errorf("service unavailable")
	}
	zoneID := strings.TrimSpace(req.CloudflareZoneID)
	zoneName := normalizeZoneName(req.ZoneName)
	if zoneID == "" {
		return WildcardSSLResponse{}, fmt.Errorf("cloudflare_zone_id is required")
	}
	if zoneName == "" {
		return WildcardSSLResponse{}, fmt.Errorf("zone_name is required")
	}
	cf := dh.cloudflareAPI()
	if cf == nil {
		return WildcardSSLResponse{}, fmt.Errorf("cloudflare api not configured")
	}

	poolID, err := dh.resolveDomainPoolID(ctx, req.PoolID)
	if err != nil {
		return WildcardSSLResponse{}, err
	}
	wildcardHost := "*." + zoneName

	rowID := uuid.New()
	err = dh.host.Pool().QueryRow(ctx, `
		INSERT INTO domain_wildcard_ssl (
			id, pool_id, cloudflare_zone_id, zone_name, wildcard_hostname, include_apex, acme_state
		) VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		ON CONFLICT (pool_id, zone_name) DO UPDATE SET
			cloudflare_zone_id = EXCLUDED.cloudflare_zone_id,
			wildcard_hostname = EXCLUDED.wildcard_hostname,
			include_apex = EXCLUDED.include_apex,
			acme_state = 'pending',
			last_error = '',
			updated_at = now()
		RETURNING id`, rowID, poolID, zoneID, zoneName, wildcardHost, req.IncludeApex).Scan(&rowID)
	if err != nil {
		return WildcardSSLResponse{}, err
	}

	email := acmeEmailFromConfig(dh.host.Config())
	directory := acmeDirectoryFromConfig(dh.host.Config())
	issueReq := wildcardIssueRequest{
		ZoneName:    zoneName,
		IncludeApex: req.IncludeApex,
		Email:       email,
		ZoneID:      zoneID,
		Directory:   directory,
	}
	issued, issueErr := defaultWildcardIssuer.Issue(ctx, cf, issueReq)

	resp := WildcardSSLResponse{
		ID:                rowID.String(),
		PoolID:            poolID.String(),
		WildcardHostname:  wildcardHost,
		CloudflareProxied: true,
	}
	if issueErr != nil {
		resp.AcmeState = "failed"
		resp.Message = issueErr.Error()
		_, _ = dh.host.Pool().Exec(ctx, `
			UPDATE domain_wildcard_ssl
			SET acme_state = 'failed', last_error = $2, updated_at = now()
			WHERE pool_id = $1 AND zone_name = $3`, poolID, issueErr.Error(), zoneName)
		return resp, nil
	}

	keyEnc, err := encryptDomainSSLKey(dh.host.Config(), issued.KeyPEM)
	if err != nil {
		return WildcardSSLResponse{}, err
	}
	var storedID uuid.UUID
	err = dh.host.Pool().QueryRow(ctx, `
		UPDATE domain_wildcard_ssl
		SET acme_state = 'valid',
			cert_pem = $3,
			key_pem_encrypted = $4,
			ssl_not_after = $5,
			last_error = '',
			updated_at = now()
		WHERE pool_id = $1 AND zone_name = $2
		RETURNING id`, poolID, zoneName, string(issued.CertPEM), keyEnc, issued.NotAfter).Scan(&storedID)
	if err != nil {
		return WildcardSSLResponse{}, err
	}

	_, err = dh.host.Pool().Exec(ctx, `
		INSERT INTO domain_health_status (hostname, role, ssl_status, ssl_not_after)
		VALUES ($1, 'custom', 'valid', $2)
		ON CONFLICT (hostname) DO UPDATE SET
			ssl_status = 'valid',
			ssl_not_after = EXCLUDED.ssl_not_after,
			updated_at = now()`, wildcardHost, issued.NotAfter)
	if err != nil {
		return WildcardSSLResponse{}, err
	}

	resp.ID = storedID.String()
	resp.AcmeState = "valid"
	resp.SSLNotAfter = &issued.NotAfter
	resp.Message = "wildcard certificate issued"
	return resp, nil
}

func (dh *DomainHealth) renewWildcardCerts(ctx context.Context) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return
	}
	rows, err := dh.host.Pool().Query(ctx, `
		SELECT id, pool_id, cloudflare_zone_id, zone_name, include_apex
		FROM domain_wildcard_ssl
		WHERE acme_state = 'valid'
		  AND ssl_not_after IS NOT NULL
		  AND ssl_not_after < now() + interval '30 days'`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, poolID uuid.UUID
		var zoneID, zoneName string
		var includeApex bool
		if err := rows.Scan(&id, &poolID, &zoneID, &zoneName, &includeApex); err != nil {
			continue
		}
		_, err := dh.SetupWildcardSSL(ctx, WildcardSSLRequest{
			CloudflareZoneID: zoneID,
			ZoneName:         zoneName,
			PoolID:           &poolID,
			IncludeApex:      includeApex,
		})
		if err != nil {
			recordDomainSSLRenew("failed")
			continue
		}
		recordDomainSSLRenew("success")
	}
}

func (dh *DomainHealth) cloudflareAPI() CloudflareAPI {
	if dh == nil || dh.host == nil {
		return nil
	}
	client := dh.host.CloudflareClient()
	if client == nil {
		return nil
	}
	if api, ok := client.(CloudflareAPI); ok {
		return api
	}
	return &cloudflareClientAdapter{inner: client}
}

type cloudflareClientAdapter struct {
	inner DomainCloudflareClient
}

func (a *cloudflareClientAdapter) ListZones(ctx context.Context) ([]CloudflareZone, error) {
	return nil, errors.New("cloudflare list zones not available on narrow client")
}

func (a *cloudflareClientAdapter) CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, proxied bool) (string, error) {
	return a.inner.CreateDNSRecord(ctx, zoneID, name, recordType, content, proxied)
}

func (a *cloudflareClientAdapter) UpsertTXTRecord(ctx context.Context, zoneID, name, content string) (string, error) {
	return a.inner.UpsertTXTRecord(ctx, zoneID, name, content)
}

func (a *cloudflareClientAdapter) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	return a.inner.DeleteDNSRecord(ctx, zoneID, recordID)
}

func (a *cloudflareClientAdapter) ZoneSSLStatus(ctx context.Context, zoneID string) (string, error) {
	return a.inner.ZoneSSLStatus(ctx, zoneID)
}

func acmeEmailFromConfig(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Management.DomainSSLAcmeEmail)
}

func acmeDirectoryFromConfig(cfg *config.Config) string {
	if cfg == nil {
		return acmeLEProductionDirectory
	}
	if cfg.Management.DomainWildcardSSLStaging {
		return acmeLEStagingDirectory
	}
	dir := strings.TrimSpace(cfg.Management.ACMEDirectoryURL)
	if dir == "" {
		return acmeLEProductionDirectory
	}
	return dir
}

func encryptDomainSSLKey(cfg *config.Config, keyPEM []byte) ([]byte, error) {
	storageKey := domainSSLStorageKey(cfg)
	if len(storageKey) == 0 {
		return keyPEM, nil
	}
	block, err := aes.NewCipher(storageKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, keyPEM, nil), nil
}

func domainSSLStorageKey(cfg *config.Config) []byte {
	if cfg == nil {
		return nil
	}
	raw := strings.TrimSpace(string(cfg.TokenSymmetricKey))
	if len(raw) < 32 {
		return nil
	}
	return []byte(raw[:32])
}

func (dh *DomainHealth) enrichDomainHealthDTO(ctx context.Context, dto *DomainHealthDTO) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil || dto == nil {
		return
	}
	var acmeState pgtype.Text
	var proxied pgtype.Bool
	var poolID pgtype.UUID
	var wildcardZone pgtype.Text
	var poolStatus pgtype.Text
	var zoneID string
	err := dh.host.Pool().QueryRow(ctx, `
		SELECT status, pool_id, COALESCE(cloudflare_zone_id, '')
		FROM domain_pool_domains
		WHERE hostname = $1`, dto.Hostname).Scan(&poolStatus, &poolID, &zoneID)
	if err == nil {
		if poolStatus.Valid {
			dto.PoolStatus = poolStatus.String
		}
		if poolID.Valid {
			id := uuid.UUID(poolID.Bytes)
			dto.PoolID = id.String()
		}
		if zoneID != "" {
			dto.CloudflareProxied = true
		}
	}
	err = dh.host.Pool().QueryRow(ctx, `
		SELECT w.acme_state, w.cloudflare_proxied, w.pool_id, w.zone_name
		FROM domain_wildcard_ssl w
		WHERE w.wildcard_hostname = $1
		   OR $1 LIKE '%.' || w.zone_name
		ORDER BY w.updated_at DESC
		LIMIT 1`, dto.Hostname).Scan(&acmeState, &proxied, &poolID, &wildcardZone)
	if errors.Is(err, pgx.ErrNoRows) {
		return
	}
	if err != nil {
		return
	}
	if acmeState.Valid {
		dto.AcmeState = acmeState.String
	}
	if proxied.Valid {
		dto.CloudflareProxied = proxied.Bool
	}
	if poolID.Valid {
		id := uuid.UUID(poolID.Bytes)
		dto.PoolID = id.String()
	}
	if wildcardZone.Valid {
		dto.WildcardZone = wildcardZone.String
	}
}

func (dh *DomainHealth) tlsAllowedByWildcard(ctx context.Context, host string) (bool, error) {
	rows, err := dh.host.Pool().Query(ctx, `
		SELECT zone_name, include_apex
		FROM domain_wildcard_ssl
		WHERE acme_state = 'valid'`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var zoneName string
		var includeApex bool
		if err := rows.Scan(&zoneName, &includeApex); err != nil {
			return false, err
		}
		if hostnameMatchesWildcardZone(host, zoneName, includeApex) {
			return true, nil
		}
	}
	return false, nil
}
