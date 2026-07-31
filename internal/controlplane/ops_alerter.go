package controlplane

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"espx/internal/config"
	"espx/internal/metrics"
	"espx/internal/notifier"
	"espx/pkg/branding"
)

type OpsAlerter struct {
	client             *NotifierClient
	provider           string
	recipient          string
	broadcastProviders []string
	cooldown           time.Duration
	outboxStuckSec     int
	lastSent           sync.Map
	enqueueFailures    atomic.Int64
}

func NewOpsAlerter(client *NotifierClient, cfg *config.Config) *OpsAlerter {
	if client == nil || cfg == nil || !cfg.OpsAlertsEnabled() {
		return nil
	}
	provider, recipient, ok := resolveOpsAlertTarget(cfg)
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
		client:             client,
		provider:           provider,
		recipient:          recipient,
		broadcastProviders: resolveBroadcastProviders(cfg),
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

func (a *OpsAlerter) sendAsync(key, title, body string, broadcast bool) {
	if a == nil || !a.shouldSend(key) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := a.dispatch(ctx, key, title, body, broadcast); err != nil {
			failures := a.enqueueFailures.Add(1)
			metrics.ManagementOpsAlertEnqueueFailuresTotal.Inc()
			slog.Warn("ops alert enqueue failed", "key", key, "error", err, "consecutive_failures", failures)
			if failures == 1 || failures%5 == 0 {
				metaTitle := branding.AlertTitle("ops alert enqueue failing")
				metaBody := fmt.Sprintf(
					"<b>Notifier enqueue failures</b>\nConsecutive failures: %d\nLast key: %s\nError: %v",
					failures, key, err,
				)
				if a.shouldSend("notifier:enqueue") {
					if metaErr := a.dispatch(ctx, "notifier:enqueue", metaTitle, metaBody, true); metaErr != nil {
						slog.Warn("ops meta alert enqueue failed", "error", metaErr)
					}
				}
			}
			return
		}
		a.enqueueFailures.Store(0)
	}()
}

func (a *OpsAlerter) dispatch(ctx context.Context, key, title, body string, broadcast bool) error {
	if a == nil {
		return fmt.Errorf("ops alerter not configured")
	}
	target := notifier.OpsAlertTarget{Provider: a.provider, Recipient: a.recipient}
	return enqueueOpsNotification(ctx, a.client, target, title, body, key, broadcast, a.broadcastProviders)
}

func (a *OpsAlerter) enqueueNotification(ctx context.Context, key, title, body string, broadcast bool) error {
	return a.dispatch(ctx, key, title, body, broadcast)
}

func (a *OpsAlerter) AlertReconDiscrepancy(runID int64, discrepancies int, totalDelta int64, period string) {
	if a == nil || discrepancies <= 0 {
		return
	}
	key := fmt.Sprintf("recon:run:%d", runID)
	title := branding.AlertTitle("recon discrepancy")
	body := fmt.Sprintf(
		"<b>Recon discrepancy</b>\nPeriod: %s\nRun #%d\nCampaigns adjusted: %d\nTotal delta (micro): %d",
		period, runID, discrepancies, totalDelta,
	)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertReconDiscrepancyUnresolved(runID int64, unresolved int, totalDelta int64, period string, oldest time.Time) {
	if a == nil || unresolved <= 0 {
		return
	}
	key := fmt.Sprintf("recon:unresolved:%d", runID)
	title := branding.AlertTitle("unreconciled budget drift")
	body := fmt.Sprintf(
		"<b>Unresolved recon discrepancy</b>\nPeriod: %s\nRun #%d\nUnresolved campaigns: %d\nTotal |delta| (micro): %d\nOldest since: %s",
		period, runID, unresolved, totalDelta, oldest.UTC().Format(time.RFC3339),
	)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertRedisShardUnhealthy(shardIdx int, err error) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("redis:shard:%d", shardIdx)
	title := fmt.Sprintf("%s: Redis shard %d unreachable", branding.ProductName(), shardIdx)
	body := fmt.Sprintf(
		"<b>Redis shard unhealthy</b>\nShard: %d\nError: %v\nStuck quota reservations were released.",
		shardIdx, err,
	)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertSlotMapMigrating(version int32, slots []int16, targetShard int16) {
	if a == nil || len(slots) == 0 {
		return
	}
	key := fmt.Sprintf("migration:mark:%d:%d:%s", version, targetShard, formatSlotIDs(slots))
	title := branding.AlertTitle("slot map migration started")
	body := fmt.Sprintf(
		"<b>Slot map migration</b>\nVersion: %d\nTarget shard: %d\nSlots (%d): %s\nNext: copy data, then activate.",
		version, targetShard, len(slots), formatSlotIDs(slots),
	)
	a.sendAsync(key, title, body, false)
}

func (a *OpsAlerter) AlertDrainStuck(version int32, slot int16, state, lastError string, updatedAt time.Time) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("drain:%d:%d:%s", version, slot, state)
	title := branding.AlertTitle("slot migration drain stuck")
	body := fmt.Sprintf(
		"<b>Drain stuck</b>\nVersion: %d\nSlot: %d\nState: %s\nSince: %s\nError: %s",
		version, slot, state, updatedAt.UTC().Format(time.RFC3339), lastError,
	)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertBlacklistJanitorFailed(err error) {
	if a == nil || err == nil {
		return
	}
	key := "blacklist:janitor:scan"
	title := branding.AlertTitle("blacklist janitor failed")
	body := fmt.Sprintf("<b>Blacklist janitor scan failed</b>\nError: %v", err)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertOutboxStuck(pending int64, oldestSeconds float64) {
	if a == nil || pending <= 0 {
		return
	}
	key := fmt.Sprintf("outbox:stuck:%d", int64(oldestSeconds)/60)
	title := branding.AlertTitle("outbox backlog stale")
	body := fmt.Sprintf(
		"<b>Outbox backlog stale</b>\nPending events: %d\nOldest pending age (s): %.0f\nHot-path Redis may drift from Postgres.",
		pending, oldestSeconds,
	)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) OutboxStuckThresholdSec() int {
	if a == nil || a.outboxStuckSec <= 0 {
		return 120
	}
	return a.outboxStuckSec
}

func (a *OpsAlerter) AlertSlotMigrationComplete(version int32) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("migration:complete:%d", version)
	title := branding.AlertTitle("slot migration cutover complete")
	body := fmt.Sprintf("<b>Slot migration complete</b>\nActive version: %d\nPost-cutover R5 verification passed.", version)
	a.sendAsync(key, title, body, false)
}

func (a *OpsAlerter) AlertLedgerDrift(customerID string, driftErr error) {
	if a == nil || driftErr == nil {
		return
	}
	key := fmt.Sprintf("billing:drift:%s", customerID)
	title := branding.AlertTitle("billing ledger drift")
	body := fmt.Sprintf("<b>Ledger invariant failed</b>\nCustomer: %s\nError: %v", customerID, driftErr)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertCHEmergencyDrop(table, partition string, diskUsedPct float64, thresholdPct int) {
	if a == nil {
		return
	}
	key := fmt.Sprintf("ch:emergency:%s:%s", table, partition)
	title := branding.AlertTitle("ClickHouse emergency partition drop")
	body := fmt.Sprintf(
		"<b>CH emergency drop</b>\nTable: %s\nPartition: %s\nDisk used: %.1f%%\nThreshold: %d%%\nReview retention and ingest volume.",
		table, partition, diskUsedPct, thresholdPct,
	)
	a.sendAsync(key, title, body, true)
}

func (a *OpsAlerter) AlertSlotMigrationError(stage string, err error) {
	if a == nil || err == nil {
		return
	}
	key := fmt.Sprintf("migration:tick:%s", stage)
	title := fmt.Sprintf("%s: slot migration %s failed", branding.ProductName(), stage)
	body := fmt.Sprintf("<b>Slot migration error</b>\nStage: %s\nError: %v", stage, err)
	a.sendAsync(key, title, body, true)
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
