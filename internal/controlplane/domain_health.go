package controlplane

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/pkg/domainhealth"
	"github.com/bidshard/ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type domainTarget struct {
	Hostname string
	Role     string
}

func (s *Service) ListDomainHealth(ctx context.Context) ([]adminapi.DomainHealthDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	if err := s.syncDomainTargets(ctx); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT hostname, role, health_status, ssl_status, ssl_not_after,
		       http_status, probe_latency_ms, probe_detail, last_probe_at, updated_at
		FROM domain_health_status
		ORDER BY role, hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []adminapi.DomainHealthDTO
	for rows.Next() {
		dto, err := scanDomainHealth(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *Service) AddCustomDomain(ctx context.Context, hostname string) (adminapi.DomainHealthDTO, error) {
	if s == nil || s.pool == nil {
		return adminapi.DomainHealthDTO{}, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return adminapi.DomainHealthDTO{}, fmt.Errorf("hostname is required")
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO domain_health_status (hostname, role)
		VALUES ($1, 'custom')
		ON CONFLICT (hostname) DO UPDATE SET updated_at = now()
		RETURNING hostname, role, health_status, ssl_status, ssl_not_after,
		          http_status, probe_latency_ms, probe_detail, last_probe_at, updated_at`,
		host)
	return scanDomainHealth(row)
}

func (s *Service) DeleteCustomDomain(ctx context.Context, hostname string) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM domain_health_status
		WHERE hostname = $1 AND role = 'custom'`, host)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("domain not found or not removable")
	}
	return nil
}

func (s *Service) ProbeDomainNow(ctx context.Context, hostname string) (adminapi.DomainHealthDTO, error) {
	if s == nil || s.pool == nil {
		return adminapi.DomainHealthDTO{}, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	var role string
	err := s.pool.QueryRow(ctx, `SELECT role FROM domain_health_status WHERE hostname = $1`, host).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminapi.DomainHealthDTO{}, fmt.Errorf("domain not found")
		}
		return adminapi.DomainHealthDTO{}, err
	}
	if err := s.probeAndStore(ctx, domainTarget{Hostname: host, Role: role}); err != nil {
		return adminapi.DomainHealthDTO{}, err
	}
	row := s.pool.QueryRow(ctx, `
		SELECT hostname, role, health_status, ssl_status, ssl_not_after,
		       http_status, probe_latency_ms, probe_detail, last_probe_at, updated_at
		FROM domain_health_status WHERE hostname = $1`, host)
	return scanDomainHealth(row)
}

func (s *Service) SetupDomainSSL(ctx context.Context, hostname string) (adminapi.DomainSSLSetupResult, error) {
	if s == nil || s.cfg == nil {
		return adminapi.DomainSSLSetupResult{}, fmt.Errorf("service unavailable")
	}
	if !s.cfg.Management.DomainSSLSetupEnabled {
		return adminapi.DomainSSLSetupResult{}, fmt.Errorf("ssl setup disabled on this deployment")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return adminapi.DomainSSLSetupResult{}, fmt.Errorf("hostname is required")
	}
	var exists int
	err := s.pool.QueryRow(ctx, `SELECT 1 FROM domain_health_status WHERE hostname = $1`, host).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminapi.DomainSSLSetupResult{}, fmt.Errorf("domain not registered")
		}
		return adminapi.DomainSSLSetupResult{}, err
	}

	script := s.cfg.Management.DomainSSLSetupScript
	if script == "" {
		script = filepath.Join("scripts", "install", "setup_domain_ssl.sh")
	}
	if _, err := os.Stat(script); err != nil {
		return adminapi.DomainSSLSetupResult{}, fmt.Errorf("ssl setup script not found: %s", script)
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, "/bin/bash", script, host)
	cmd.Dir = config.InstallRootFromEnv()
	if cmd.Dir == "" {
		cmd.Dir = "."
	}
	cmd.Env = append(os.Environ(),
		"DOMAIN="+host,
		"CADDY_ACME_EMAIL="+s.cfg.Management.DomainSSLAcmeEmail,
	)
	out, err := cmd.CombinedOutput()
	result := adminapi.DomainSSLSetupResult{
		Hostname: host,
		Output:   strings.TrimSpace(string(out)),
	}
	if err != nil {
		result.Status = "failed"
		result.Message = err.Error()
		return result, nil
	}
	result.Status = "ok"
	result.Message = "certificate setup completed"
	_, _ = s.ProbeDomainNow(ctx, host)
	return result, nil
}

func (s *Service) syncDomainTargets(ctx context.Context) error {
	targets, err := s.discoverDomainTargets(ctx)
	if err != nil {
		return err
	}
	for _, t := range targets {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO domain_health_status (hostname, role)
			VALUES ($1, $2)
			ON CONFLICT (hostname) DO UPDATE
			SET role = EXCLUDED.role, updated_at = now()
			WHERE domain_health_status.role IN ('tracking', 'admin')`,
			t.Hostname, t.Role)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) discoverDomainTargets(ctx context.Context) ([]domainTarget, error) {
	var out []domainTarget
	if s == nil {
		return out, nil
	}
	cfg, _, err := s.GetPlatformConfig(ctx)
	if err != nil {
		return nil, err
	}
	if host := platformconfig.ResolveHost(cfg.TrackingDomain); host != "" {
		out = append(out, domainTarget{Hostname: host, Role: "tracking"})
	}
	if s.cfg != nil {
		if host := platformconfig.ResolveHost(s.cfg.AdminDomain); host != "" {
			out = append(out, domainTarget{Hostname: host, Role: "admin"})
		}
	}
	return out, nil
}

type DomainHealthWorker struct {
	svc      *Service
	interval time.Duration
}

func NewDomainHealthWorker(svc *Service, interval time.Duration) *DomainHealthWorker {
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}
	return &DomainHealthWorker{svc: svc, interval: interval}
}

func (s *Service) StartDomainHealthWorker(ctx context.Context, interval time.Duration) {
	if s == nil || s.cfg == nil || !s.cfg.Management.DomainHealthEnabled {
		return
	}
	w := NewDomainHealthWorker(s, interval)
	s.StartBackgroundWorker(func() {
		w.Start(ctx)
	})
	slog.Info("domain health worker enabled", "interval", w.interval)
}

func (w *DomainHealthWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	w.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx)
		}
	}
}

func (w *DomainHealthWorker) tick(ctx context.Context) {
	if w == nil || w.svc == nil || w.svc.pool == nil {
		return
	}
	if err := w.svc.syncDomainTargets(ctx); err != nil {
		slog.Error("domain health: sync targets", "err", err)
		return
	}
	rows, err := w.svc.pool.Query(ctx, `SELECT hostname, role FROM domain_health_status`)
	if err != nil {
		slog.Error("domain health: list domains", "err", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var host, role string
		if err := rows.Scan(&host, &role); err != nil {
			slog.Error("domain health: scan", "err", err)
			continue
		}
		if err := w.svc.probeAndStore(ctx, domainTarget{Hostname: host, Role: role}); err != nil {
			slog.Error("domain health: probe", "host", host, "err", err)
		}
	}
}

func (s *Service) probeAndStore(ctx context.Context, target domainTarget) error {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	res := domainhealth.Probe(probeCtx, target.Hostname, target.Role)
	now := time.Now().UTC()

	var sslNotAfter pgtype.Timestamptz
	if res.SSLNotAfter != nil {
		sslNotAfter = pgtype.Timestamptz{Time: *res.SSLNotAfter, Valid: true}
	}
	var httpStatus pgtype.Int4
	if res.HTTPStatus > 0 {
		httpStatus = pgtype.Int4{Int32: int32(res.HTTPStatus), Valid: true}
	}
	_, err := s.pool.Exec(ctx, `
		UPDATE domain_health_status
		SET health_status = $2,
		    ssl_status = $3,
		    ssl_not_after = $4,
		    http_status = $5,
		    probe_latency_ms = $6,
		    probe_detail = $7,
		    last_probe_at = $8,
		    updated_at = now()
		WHERE hostname = $1`,
		target.Hostname,
		res.HealthStatus,
		res.SSLStatus,
		sslNotAfter,
		httpStatus,
		int(res.ProbeLatencyMs),
		res.ProbeDetail,
		now,
	)
	return err
}

type domainHealthScanner interface {
	Scan(dest ...any) error
}

func scanDomainHealth(row domainHealthScanner) (adminapi.DomainHealthDTO, error) {
	var dto adminapi.DomainHealthDTO
	var sslNotAfter pgtype.Timestamptz
	var httpStatus pgtype.Int4
	var probeLatency pgtype.Int4
	var lastProbe pgtype.Timestamptz
	if err := row.Scan(
		&dto.Hostname, &dto.Role, &dto.HealthStatus, &dto.SSLStatus, &sslNotAfter,
		&httpStatus, &probeLatency, &dto.ProbeDetail, &lastProbe, &dto.UpdatedAt,
	); err != nil {
		return adminapi.DomainHealthDTO{}, err
	}
	if sslNotAfter.Valid {
		t := sslNotAfter.Time
		dto.SSLNotAfter = &t
	}
	if httpStatus.Valid {
		v := int(httpStatus.Int32)
		dto.HTTPStatus = &v
	}
	if probeLatency.Valid {
		v := int(probeLatency.Int32)
		dto.ProbeLatencyMs = &v
	}
	if lastProbe.Valid {
		t := lastProbe.Time
		dto.LastProbeAt = &t
	}
	return dto, nil
}

// IsTLSAllowed reports whether Caddy may issue an on-demand certificate for hostname.
// Only tracking and custom buyer domains are permitted (admin role excluded).
func (s *Service) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	if s == nil || s.pool == nil {
		return false, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return false, nil
	}
	var one int
	err := s.pool.QueryRow(ctx, `
		SELECT 1 FROM domain_health_status
		WHERE hostname = $1 AND role IN ('custom', 'tracking')
		LIMIT 1`, host).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
