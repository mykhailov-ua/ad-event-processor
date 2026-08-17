package controlplane

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"

	"github.com/google/uuid"
)

func (r *opsReader) ListDomainRotation(ctx context.Context) (adminapi.DomainRotationListResult, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return adminapi.DomainRotationListResult{}, fmt.Errorf("postgres pool not configured")
	}
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT d.hostname, d.role, d.health_status, COALESCE(d.ssl_status, ''),
		       dpd.pool_id, COALESCE(dpd.status, ''),
		       COALESCE((
		           SELECT COUNT(*)::bigint FROM campaigns c
		           WHERE c.domain_pool_id = dpd.pool_id AND c.dmr_enabled = true AND c.deleted_at IS NULL
		       ), 0),
		       COALESCE((
		           SELECT COUNT(*)::bigint FROM campaigns c
		           WHERE c.domain_pool_id = dpd.pool_id AND c.status = 'active' AND c.deleted_at IS NULL
		       ), 0)
		FROM domain_health_status d
		LEFT JOIN domain_pool_domains dpd ON dpd.hostname = d.hostname
		WHERE d.role IN ('tracking', 'admin')
		ORDER BY d.hostname ASC`)
	if err != nil {
		return adminapi.DomainRotationListResult{}, err
	}
	defer rows.Close()

	var hosts []adminapi.DomainRotationHostDTO
	for rows.Next() {
		var (
			hostname     string
			role         string
			healthStatus string
			sslStatus    string
			poolID       *uuid.UUID
			poolStatus   string
			dmrCount     int64
			activeCount  int64
		)
		if err := rows.Scan(&hostname, &role, &healthStatus, &sslStatus, &poolID, &poolStatus, &dmrCount, &activeCount); err != nil {
			return adminapi.DomainRotationListResult{}, err
		}
		row := adminapi.DomainRotationHostDTO{
			Hostname:            hostname,
			Role:                role,
			HealthStatus:        healthStatus,
			SSLStatus:           sslStatus,
			PoolDomainStatus:    poolStatus,
			DmrCampaignCount:    dmrCount,
			ActiveCampaignCount: activeCount,
		}
		if poolID != nil && *poolID != uuid.Nil {
			row.PoolID = poolID.String()
		}
		hosts = append(hosts, row)
	}
	if err := rows.Err(); err != nil {
		return adminapi.DomainRotationListResult{}, err
	}
	if hosts == nil {
		hosts = []adminapi.DomainRotationHostDTO{}
	}
	return adminapi.DomainRotationListResult{Hosts: hosts}, nil
}
