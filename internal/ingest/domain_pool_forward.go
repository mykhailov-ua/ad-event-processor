package ingest

import (
	"ad-event-processor/internal/ingest/pool"
)

type DomainPoolTable = pool.Table

var (
	NewDomainPoolTable = pool.NewTable
	NewDomainPoolSync  = pool.NewSync
)
