package opsadmin

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/notify"
	"ad-event-processor/pkg/branding"
)

type OpsAlerter struct {
	api                notify.NotifierAPI
	provider           string
	recipient          string
	broadcastProviders []string
	cooldown           time.Duration
	outboxStuckSec     int
	lastSent           sync.Map
	enqueueFailures    atomic.Int64
	wg                 sync.WaitGroup
}

func NewOpsAlerter(api notify.NotifierAPI, cfg *config.Config) *OpsAlerter {
	if api == nil || cfg == nil || !cfg.OpsAlertsEnabled() {
		return nil
	}
	provider, recipient, ok := ResolveOpsAlertTarget(cfg)
	if !ok {
		return nil
	}
	cooldownSec := cfg.Management.OpsAlertCooldownSec
	if cooldownSec <= 0 {
		cooldownSec = 300
	}
	outboxStuckSec := cfg.Management.OpsAlertOutboxStuckSec
	if outboxStuckSec <= 0 {
		outboxStuckSec = 120
	}
	return &OpsAlerter{
		api:                api,
		provider:           provider,
		recipient:          recipient,
		broadcastProviders: ResolveBroadcastProviders(cfg),
		cooldown:           time.Duration(cooldownSec) * time.Second,
		outboxStuckSec:     outboxStuckSec,
	}
}

func (a *OpsAlerter) shouldSend(key string) bool {
	if a == nil {
		return false
	}
	now := time.Now()
	if v, ok := a.lastSent.Load(key); ok {
		if now.Sub(v.(time.Time)) < a.cooldown {
			return false
		}
	}
	a.lastSent.Store(key, now)
	return true
}

func (a *OpsAlerter) sendAsync(ctx context.Context, key, title, body string, broadcast bool) {
	if a == nil || !a.shouldSend(key) {
		return
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := a.dispatch(sendCtx, key, title, body, broadcast); err != nil {
			failures := a.enqueueFailures.Add(1)
			metrics.IncControlOpsAlertEnqueueFailures()
			slog.Warn("ops alert enqueue failed", "key", key, "error", err, "consecutive_failures", failures)
			if failures == 1 || failures%5 == 0 {
				metaTitle := branding.AlertTitle("ops alert enqueue failing")
				metaBody := fmt.Sprintf(
					"<b>Notifier enqueue failures</b>\nConsecutive failures: %d\nLast key: %s\nError: %v",
					failures, key, err,
				)
				if a.shouldSend("notifier:enqueue") {
					if metaErr := a.dispatch(sendCtx, "notifier:enqueue", metaTitle, metaBody, true); metaErr != nil {
						slog.Warn("ops meta alert enqueue failed", "error", metaErr)
					}
				}
			}
			return
		}
		a.enqueueFailures.Store(0)
	}()
}

func (a *OpsAlerter) Drain() {
	if a != nil {
		a.wg.Wait()
	}
}

func (a *OpsAlerter) dispatch(ctx context.Context, key, title, body string, broadcast bool) error {
	if a == nil {
		return fmt.Errorf("ops alerter not configured")
	}
	target := notify.OpsAlertTarget{Provider: a.provider, Recipient: a.recipient}
	return EnqueueOpsNotification(ctx, a.api, target, title, body, key, broadcast, a.broadcastProviders)
}

func (a *OpsAlerter) enqueueNotification(ctx context.Context, key, title, body string, broadcast bool) error {
	return a.dispatch(ctx, key, title, body, broadcast)
}

func (a *OpsAlerter) AlertReconDiscrepancy(ctx context.Context, runID int64, discrepancies int, totalDelta int64, period string) {
	if a == nil || discrepancies <= 0 {
		return
	}
	key := fmt.Sprintf("recon:run:%d", runID)
	title := branding.AlertTitle("recon discrepancy")
	body := fmt.Sprintf(
		"<b>Recon discrepancy</b>\nPeriod: %s\nRun #%d\nCampaigns adjusted: %d\nTotal delta (micro): %d",
		period, runID, discrepancies, totalDelta,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertReconDiscrepancyUnresolved(ctx context.Context, runID int64, unresolved int, totalDelta int64, period string, oldest time.Time) {
	if a == nil || unresolved <= 0 {
		return
	}
	key := fmt.Sprintf("recon:unresolved:%d", runID)
	title := branding.AlertTitle("unreconciled budget drift")
	body := fmt.Sprintf(
		"<b>Unresolved recon discrepancy</b>\nPeriod: %s\nRun #%d\nUnresolved campaigns: %d\nTotal |delta| (micro): %d\nOldest since: %s",
		period, runID, unresolved, totalDelta, oldest.UTC().Format(time.RFC3339),
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertRedisShardUnhealthy(ctx context.Context, shardIdx int, err error) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("redis:shard:%d", shardIdx)
	title := fmt.Sprintf("%s: Redis shard %d unreachable", branding.ProductName(), shardIdx)
	body := fmt.Sprintf(
		"<b>Redis shard unhealthy</b>\nShard: %d\nError: %v\nStuck quota reservations were released.",
		shardIdx, err,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertSlotMapMigrating(ctx context.Context, version int32, slots []int16, targetShard int16) {
	if a == nil || len(slots) == 0 {
		return
	}
	key := fmt.Sprintf("migration:mark:%d:%d:%s", version, targetShard, formatSlotIDs(slots))
	title := branding.AlertTitle("slot map migration started")
	body := fmt.Sprintf(
		"<b>Slot map migration</b>\nVersion: %d\nTarget shard: %d\nSlots (%d): %s\nNext: copy data, then activate.",
		version, targetShard, len(slots), formatSlotIDs(slots),
	)
	a.sendAsync(ctx, key, title, body, false)
}

func (a *OpsAlerter) AlertDrainStuck(ctx context.Context, version int32, slot int16, state, lastError string, updatedAt time.Time) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("drain:%d:%d:%s", version, slot, state)
	title := branding.AlertTitle("slot migration drain stuck")
	body := fmt.Sprintf(
		"<b>Drain stuck</b>\nVersion: %d\nSlot: %d\nState: %s\nSince: %s\nError: %s",
		version, slot, state, updatedAt.UTC().Format(time.RFC3339), lastError,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertBlacklistJanitorFailed(ctx context.Context, err error) {
	if a == nil || err == nil {
		return
	}
	key := "blacklist:janitor:scan"
	title := branding.AlertTitle("blacklist janitor failed")
	body := fmt.Sprintf("<b>Blacklist janitor scan failed</b>\nError: %v", err)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertOutboxStuck(ctx context.Context, pending int64, oldestSeconds float64) {
	if a == nil || pending <= 0 {
		return
	}
	key := fmt.Sprintf("outbox:stuck:%d", int64(oldestSeconds)/60)
	title := branding.AlertTitle("outbox backlog stale")
	body := fmt.Sprintf(
		"<b>Outbox backlog stale</b>\nPending events: %d\nOldest pending age (s): %.0f\nHot-path Redis may drift from Postgres.",
		pending, oldestSeconds,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) OutboxStuckThresholdSec() int {
	if a == nil || a.outboxStuckSec <= 0 {
		return 120
	}
	return a.outboxStuckSec
}

func (a *OpsAlerter) AlertSlotMigrationComplete(ctx context.Context, version int32) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("migration:complete:%d", version)
	title := branding.AlertTitle("slot migration cutover complete")
	body := fmt.Sprintf("<b>Slot migration complete</b>\nActive version: %d\nPost-cutover R5 verification passed.", version)
	a.sendAsync(ctx, key, title, body, false)
}

func (a *OpsAlerter) AlertLedgerDrift(ctx context.Context, customerID string, driftErr error) {
	if a == nil || driftErr == nil {
		return
	}
	key := fmt.Sprintf("billing:drift:%s", customerID)
	title := branding.AlertTitle("billing ledger drift")
	body := fmt.Sprintf("<b>Ledger invariant failed</b>\nCustomer: %s\nError: %v", customerID, driftErr)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertCHEmergencyDrop(ctx context.Context, table, partition string, diskUsedPct float64, thresholdPct int) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("ch:emergency:%s:%s", table, partition)
	title := branding.AlertTitle("ClickHouse emergency partition drop")
	body := fmt.Sprintf(
		"<b>CH emergency drop</b>\nTable: %s\nPartition: %s\nDisk used: %.1f%%\nThreshold: %d%%\nReview retention and ingest volume.",
		table, partition, diskUsedPct, thresholdPct,
	)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertSlotMigrationError(ctx context.Context, stage string, err error) {
	if a == nil || err == nil {
		return
	}
	key := fmt.Sprintf("migration:tick:%s", stage)
	title := fmt.Sprintf("%s: slot migration %s failed", branding.ProductName(), stage)
	body := fmt.Sprintf("<b>Slot migration error</b>\nStage: %s\nError: %v", stage, err)
	a.sendAsync(ctx, key, title, body, true)
}

func (a *OpsAlerter) AlertLicenseApplied(ctx context.Context, deploymentID string, validUntil time.Time, adminID string, revoked bool) {
	if a == nil {
		return
	}
	key := "license:apply:" + deploymentID
	title := fmt.Sprintf("%s: license applied", branding.ProductName())
	kind := "renewal"
	if revoked {
		kind = "revocation"
	}
	body := fmt.Sprintf(
		"<b>License %s</b>\nDeployment: %s\nValid until: %s\nAdmin: %s",
		kind, deploymentID, validUntil.UTC().Format(time.RFC3339), adminID,
	)
	a.sendAsync(ctx, key, title, body, false)
}

func formatSlotIDs(slots []int16) string {
	if len(slots) == 0 {
		return ""
	}
	const maxShown = 12
	parts := make([]string, 0, min(len(slots), maxShown))
	for i, slot := range slots {
		if i >= maxShown {
			parts = append(parts, fmt.Sprintf("+%d more", len(slots)-maxShown))
			break
		}
		parts = append(parts, strconv.Itoa(int(slot)))
	}
	return strings.Join(parts, ", ")
}
