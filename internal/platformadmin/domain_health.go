package platformadmin

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

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/domainhealth"
	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type domainTarget struct {
	Hostname string
	Role     string
}

func (dh *DomainHealth) ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	if err := dh.syncDomainTargets(ctx); err != nil {
		return nil, err
	}
	rows, err := dh.host.Pool().Query(ctx, `
		SELECT hostname, role, health_status, ssl_status, ssl_not_after,
		 http_status, probe_latency_ms, probe_detail, last_probe_at, updated_at
		FROM domain_health_status
		ORDER BY role, hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []DomainHealthDTO
	for rows.Next() {
		dto, err := scanDomainHealth(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (dh *DomainHealth) AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return DomainHealthDTO{}, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return DomainHealthDTO{}, fmt.Errorf("hostname is required")
	}
	row := dh.host.Pool().QueryRow(ctx, `
		INSERT INTO domain_health_status (hostname, role)
		VALUES ($1, 'custom')
		ON CONFLICT (hostname) DO UPDATE SET updated_at = now()
		RETURNING hostname, role, health_status, ssl_status, ssl_not_after,
		 http_status, probe_latency_ms, probe_detail, last_probe_at, updated_at`,
		host)
	return scanDomainHealth(row)
}

func (dh *DomainHealth) DeleteCustomDomain(ctx context.Context, hostname string) error {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	tag, err := dh.host.Pool().Exec(ctx, `
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

func (dh *DomainHealth) ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return DomainHealthDTO{}, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	var role string
	err := dh.host.Pool().QueryRow(ctx, `SELECT role FROM domain_health_status WHERE hostname = $1`, host).Scan(&role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DomainHealthDTO{}, fmt.Errorf("domain not found")
		}
		return DomainHealthDTO{}, err
	}
	if err := dh.probeAndStore(ctx, domainTarget{Hostname: host, Role: role}); err != nil {
		return DomainHealthDTO{}, err
	}
	row := dh.host.Pool().QueryRow(ctx, `
		SELECT hostname, role, health_status, ssl_status, ssl_not_after,
		 http_status, probe_latency_ms, probe_detail, last_probe_at, updated_at
		FROM domain_health_status WHERE hostname = $1`, host)
	return scanDomainHealth(row)
}

func (dh *DomainHealth) SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error) {
	if dh == nil || dh.host == nil {
		return DomainSSLSetupResult{}, fmt.Errorf("service unavailable")
	}
	cfg := dh.host.Config()
	if cfg == nil {
		return DomainSSLSetupResult{}, fmt.Errorf("service unavailable")
	}
	if !cfg.Management.DomainSSLSetupEnabled {
		return DomainSSLSetupResult{}, fmt.Errorf("ssl setup disabled on this deployment")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return DomainSSLSetupResult{}, fmt.Errorf("hostname is required")
	}
	var exists int
	err := dh.host.Pool().QueryRow(ctx, `SELECT 1 FROM domain_health_status WHERE hostname = $1`, host).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return DomainSSLSetupResult{}, fmt.Errorf("domain not registered")
		}
		return DomainSSLSetupResult{}, err
	}

	script := cfg.Management.DomainSSLSetupScript
	if script == "" {
		script = filepath.Join("scripts", "install", "setup_domain_ssl.sh")
	}
	if _, err := os.Stat(script); err != nil {
		return DomainSSLSetupResult{}, fmt.Errorf("ssl setup script not found: %s", script)
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
		"CADDY_ACME_EMAIL="+cfg.Management.DomainSSLAcmeEmail,
	)
	out, err := cmd.CombinedOutput()
	result := DomainSSLSetupResult{
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
	if _, err := dh.ProbeDomainNow(ctx, host); err != nil {
		result.Message = result.Message + "; health probe failed: " + err.Error()
	}
	return result, nil
}

func (dh *DomainHealth) syncDomainTargets(ctx context.Context) error {
	targets, err := dh.discoverDomainTargets(ctx)
	if err != nil {
		return err
	}
	for _, t := range targets {
		_, err := dh.host.Pool().Exec(ctx, `
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

func (dh *DomainHealth) discoverDomainTargets(ctx context.Context) ([]domainTarget, error) {
	var out []domainTarget
	if dh == nil || dh.host == nil {
		return out, nil
	}
	cfg, _, err := dh.host.GetPlatformConfig(ctx)
	if err != nil {
		return nil, err
	}
	if host := platformconfig.ResolveHost(cfg.TrackingDomain); host != "" {
		out = append(out, domainTarget{Hostname: host, Role: "tracking"})
	}
	if c := dh.host.Config(); c != nil {
		if host := platformconfig.ResolveHost(c.AdminDomain); host != "" {
			out = append(out, domainTarget{Hostname: host, Role: "admin"})
		}
	}
	return out, nil
}

type domainHealthWorker struct {
	dh       *DomainHealth
	interval time.Duration
}

func newDomainHealthWorker(dh *DomainHealth, interval time.Duration) *domainHealthWorker {
	if interval < 5*time.Minute {
		interval = 5 * time.Minute
	}
	return &domainHealthWorker{dh: dh, interval: interval}
}

func StartDomainHealthWorker(ctx context.Context, host DomainHealthHost, interval time.Duration) {
	if host == nil {
		return
	}
	cfg := host.Config()
	if cfg == nil || !cfg.Management.DomainHealthEnabled {
		return
	}
	dh := NewDomainHealth(host)
	w := newDomainHealthWorker(dh, interval)
	host.StartBackgroundWorker(func() {
		w.start(ctx)
	})
	slog.Info("domain health worker enabled", "interval", w.interval)
}

func (w *domainHealthWorker) start(ctx context.Context) {
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

func (w *domainHealthWorker) tick(ctx context.Context) {
	if w == nil || w.dh == nil || w.dh.host == nil || w.dh.host.Pool() == nil {
		return
	}
	if err := w.dh.syncDomainTargets(ctx); err != nil {
		slog.Error("domain health: sync targets", "err", err)
		return
	}
	rows, err := w.dh.host.Pool().Query(ctx, `SELECT hostname, role FROM domain_health_status`)
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
		if err := w.dh.probeAndStore(ctx, domainTarget{Hostname: host, Role: role}); err != nil {
			slog.Error("domain health: probe", "host", host, "err", err)
		}
	}
}

func (dh *DomainHealth) probeAndStore(ctx context.Context, target domainTarget) error {
	probeCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	res := domainhealth.Probe(probeCtx, target.Hostname, target.Role)

	if rep := dh.host.ReputationChecker(); rep != nil && rep.Enabled() {
		unsafe, detail, repErr := rep.Check(probeCtx, target.Hostname)
		if repErr != nil {
			slog.Warn("domain health: reputation probe failed", "host", target.Hostname, "err", repErr)
		} else {
			applyReputationToProbe(&res, unsafe, detail)
		}
	}

	now := time.Now().UTC()

	var sslNotAfter pgtype.Timestamptz
	if res.SSLNotAfter != nil {
		sslNotAfter = pgtype.Timestamptz{Time: *res.SSLNotAfter, Valid: true}
	}
	var httpStatus pgtype.Int4
	if res.HTTPStatus > 0 {
		httpStatus = pgtype.Int4{Int32: int32(res.HTTPStatus), Valid: true}
	}
	_, err := dh.host.Pool().Exec(ctx, `
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
	if err != nil {
		return err
	}
	if res.HealthStatus == "down" {
		if err := dh.markPoolDomainBanned(ctx, target.Hostname); err != nil {
			slog.Warn("domain health: ban pool domain", "host", target.Hostname, "err", err)
		}
	}
	return nil
}

func applyReputationToProbe(res *domainhealth.Result, unsafe bool, detail string) {
	if res == nil || !unsafe {
		return
	}
	res.HealthStatus = domainhealth.HealthDown
	if detail != "" {
		res.ProbeDetail = "reputation:" + detail
	} else {
		res.ProbeDetail = "reputation:unsafe"
	}
}

type domainHealthScanner interface {
	Scan(dest ...any) error
}

func scanDomainHealth(row domainHealthScanner) (DomainHealthDTO, error) {
	var dto DomainHealthDTO
	var sslNotAfter pgtype.Timestamptz
	var httpStatus pgtype.Int4
	var probeLatency pgtype.Int4
	var lastProbe pgtype.Timestamptz
	if err := row.Scan(
		&dto.Hostname, &dto.Role, &dto.HealthStatus, &dto.SSLStatus, &sslNotAfter,
		&httpStatus, &probeLatency, &dto.ProbeDetail, &lastProbe, &dto.UpdatedAt,
	); err != nil {
		return DomainHealthDTO{}, err
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

func (dh *DomainHealth) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return false, fmt.Errorf("service unavailable")
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return false, nil
	}
	var one int
	err := dh.host.Pool().QueryRow(ctx, `
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

func (dh *DomainHealth) MarkPoolDomainBanned(ctx context.Context, hostname string) error {
	return dh.markPoolDomainBanned(ctx, hostname)
}

func (dh *DomainHealth) markPoolDomainBanned(ctx context.Context, hostname string) error {
	if dh == nil || dh.host == nil || dh.host.Pool() == nil {
		return nil
	}
	host := platformconfig.ResolveHost(hostname)
	if host == "" {
		return nil
	}
	_, err := dh.host.Pool().Exec(ctx, `
		UPDATE domain_pool_domains
		SET status = 'banned', updated_at = now()
		WHERE hostname = $1 AND status <> 'banned'`, host)
	return err
}
