package controlplane

import (
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/marginguard"
)

type MarginGuardHTTPHandlers = marginguard.HTTPHandlers
type MarginGuardService = marginguard.Service
type MarginGuardActivityRow = marginguard.ActivityRow

type MarginGuardPolicy = ledger.Policy
