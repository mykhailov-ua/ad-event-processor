package domains

import (
	"context"
	"sync"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/domainhealth"
	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DomainCloudflareClient interface {
	CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, proxied bool) (string, error)
	UpsertTXTRecord(ctx context.Context, zoneID, name, content string) (string, error)
	DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error
	ZoneSSLStatus(ctx context.Context, zoneID string) (string, error)
}

type DomainHealthHost interface {
	Pool() *pgxpool.Pool
	Config() *config.Config
	GetPlatformConfig(ctx context.Context) (platformconfig.Config, bool, error)
	ReputationChecker() *domainhealth.ReputationChecker
	CloudflareClient() DomainCloudflareClient
	StartBackgroundWorker(fn func())
}

type DomainHealth struct {
	host DomainHealthHost

	bulkMu   sync.RWMutex
	bulkJobs map[string]*domainBulkJobRecord
}

func NewDomainHealth(host DomainHealthHost) *DomainHealth {
	return &DomainHealth{host: host}
}
