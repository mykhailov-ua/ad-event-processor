package payment

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/payment/db"
	"ad-event-processor/pkg/branding"
)

type SettlementFailedAlerter struct {
	client             *NotifierClient
	provider           string
	recipient          string
	broadcastProviders []string
	cooldown           time.Duration
	lastSent           sync.Map
}

func NewSettlementFailedAlerter(client *NotifierClient, cfg *config.Config) *SettlementFailedAlerter {
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
	return &SettlementFailedAlerter{
		client:             client,
		provider:           provider,
		recipient:          recipient,
		broadcastProviders: resolveBroadcastProviders(cfg),
		cooldown:           time.Duration(cooldownSec) * time.Second,
	}
}

func (a *SettlementFailedAlerter) shouldSend(paymentIntentID string) bool {
	if a == nil || paymentIntentID == "" {
		return false
	}
	now := time.Now()
	if v, ok := a.lastSent.Load(paymentIntentID); ok {
		if now.Sub(v.(time.Time)) < a.cooldown {
			return false
		}
	}
	a.lastSent.Store(paymentIntentID, now)
	return true
}

func (a *SettlementFailedAlerter) AlertPermanentFailure(ctx context.Context, outboxEvent db.PaymentPaymentOutbox, cause error) {
	if a == nil {
		return
	}
	intentID, ok := paymentIntentIDFromOutbox(outboxEvent)
	if !ok {
		return
	}
	if !a.shouldSend(intentID) {
		return
	}

	dedupKey := fmt.Sprintf("payment-settlement-failed:%s", intentID)
	title := branding.AlertTitle("payment settlement failed")
	body := formatSettlementFailedAlertBody(outboxEvent, intentID, cause)

	go func() {
		alertCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		if err := enqueueOpsNotification(alertCtx, a.client, a.provider, a.recipient, title, body, dedupKey, true, a.broadcastProviders); err != nil {
			metrics.IncControlOpsAlertEnqueueFailures()
			slog.Warn("payment settlement failed alert enqueue failed", "intent_id", intentID, "error", err)
		}
	}()
}

func paymentIntentIDFromOutbox(outboxEvent db.PaymentPaymentOutbox) (string, bool) {
	switch outboxEvent.EventType {
	case "SETTLE_BALANCE":
		return paymentIntentIDFromPayload[SettleBalancePayload](outboxEvent.Payload)
	case OutboxEventReverseBalance:
		return paymentIntentIDFromPayload[ReverseBalancePayload](outboxEvent.Payload)
	case OutboxEventApplyChargeback:
		return paymentIntentIDFromPayload[ApplyChargebackPayload](outboxEvent.Payload)
	case OutboxEventReverseChargeback:
		return paymentIntentIDFromPayload[ReverseChargebackPayload](outboxEvent.Payload)
	default:
		return "", false
	}
}

func formatSettlementFailedAlertBody(outboxEvent db.PaymentPaymentOutbox, intentID string, cause error) string {
	errText := ""
	if cause != nil {
		errText = cause.Error()
	}
	return fmt.Sprintf(
		"<b>Payment settlement failed</b>\nIntent: %s\nOutbox #%d (%s)\nError: %s",
		intentID, outboxEvent.ID, outboxEvent.EventType, errText,
	)
}
