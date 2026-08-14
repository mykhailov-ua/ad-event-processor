package payment

import (
	"fmt"
	"testing"
	"time"

	"context"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/notify"
	"github.com/bidshard/ad-event-processor/internal/payment/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPaymentOpsConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true
	cfg.Notifier.TelegramChatID = "-100123"
	return cfg
}

func TestFinancialFindingSeverity_mapping(t *testing.T) {
	t.Parallel()
	cases := []struct {
		kind db.PaymentFinancialFindingKind
		want FinancialFindingSeverity
	}{
		{db.PaymentFinancialFindingKindMISSINGLEDGERTOPUP, SeverityCritical},
		{db.PaymentFinancialFindingKindTOPUPAMOUNTMISMATCH, SeverityCritical},
		{db.PaymentFinancialFindingKindSETTLEMENTFAILEDINTENT, SeverityCritical},
		{db.PaymentFinancialFindingKindDEADOUTBOX, SeverityWarn},
		{db.PaymentFinancialFindingKindREFUNDLEDGERDRIFT, SeverityWarn},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, financialFindingSeverity(tc.kind), string(tc.kind))
	}
}

func TestFinancialReconAlerter_CooldownDedup(t *testing.T) {
	t.Parallel()
	cfg := testPaymentOpsConfig()
	cfg.Management.OpsAlertCooldownSec = 300

	alerter := NewFinancialReconAlerter(&NotifierClient{}, cfg)
	require.NotNil(t, alerter)

	if !alerter.shouldSend("payment-financial-recon:run:1") {
		t.Fatal("first send should pass")
	}
	if alerter.shouldSend("payment-financial-recon:run:1") {
		t.Fatal("second send within cooldown should be suppressed")
	}
}

func TestFinancialReconAlerter_AlertFindings_enqueuesWarnPlus(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testPaymentOpsConfig()
	alerter := NewFinancialReconAlerter(&NotifierClient{api: stub}, cfg)
	require.NotNil(t, alerter)

	summary := FinancialReconSummary{RunID: 42, IntentsChecked: 3}
	findings := []FinancialReconFinding{
		{Kind: db.PaymentFinancialFindingKindMISSINGLEDGERTOPUP, PaymentIntentID: uuid.New()},
	}

	alerter.AlertFindings(context.Background(), summary, findings)
	time.Sleep(150 * time.Millisecond)

	requests := stub.snapshot()
	require.Len(t, requests, 1)
	assert.Equal(t, notify.ProviderTelegram, requests[0].Provider)
	assert.True(t, requests[0].Broadcast)
	assert.Contains(t, requests[0].Body, "MISSING_LEDGER_TOPUP")
	assert.Equal(t, "payment-financial-recon:run:42", requests[0].DedupKey)
}

func TestFinancialReconAlerter_AlertFindings_skipsCleanRun(t *testing.T) {
	stub := &stubNotifierAPI{}
	cfg := testPaymentOpsConfig()
	alerter := NewFinancialReconAlerter(&NotifierClient{api: stub}, cfg)
	require.NotNil(t, alerter)

	alerter.AlertFindings(context.Background(), FinancialReconSummary{RunID: 1}, nil)
	time.Sleep(50 * time.Millisecond)
	assert.Empty(t, stub.snapshot())
}

func TestNewFinancialReconAlerter_DisabledWithoutRecipient(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{}
	cfg.Management.OpsAlertsEnabled = true
	if NewFinancialReconAlerter(&NotifierClient{}, cfg) != nil {
		t.Fatal("expected nil without recipient")
	}
}

func TestFormatFinancialReconAlertBody_includesKinds(t *testing.T) {
	body := formatFinancialReconAlertBody(FinancialReconSummary{RunID: 7, IntentsChecked: 2}, []FinancialReconFinding{
		{Kind: db.PaymentFinancialFindingKindDEADOUTBOX},
		{Kind: db.PaymentFinancialFindingKindDEADOUTBOX},
	})
	assert.Contains(t, body, "DEAD_OUTBOX: 2")
	assert.Contains(t, body, fmt.Sprintf("Run #%d", 7))
}
