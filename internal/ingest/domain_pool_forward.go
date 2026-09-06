package ingest

import (
	"ad-event-processor/internal/ingest/domainhosts"
)

type DomainPoolTable = domainhosts.Table

var (
	NewDomainPoolTable = domainhosts.NewTable
	NewDomainPoolSync  = domainhosts.NewSync
)
