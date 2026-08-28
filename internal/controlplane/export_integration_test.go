package controlplane

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJobRunner_ExportLedgerNonZeroBytes(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	ctx := context.Background()
	custID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO customers (id, name, balance, currency)
		VALUES ($1, 'Billing export', 0, 'USD')`, domain.ToUUID(custID))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO balance_ledger (customer_id, amount, type, idempotency_hash)
		VALUES ($1, 1500000, 'TOPUP', 'billing-export-held-out-1')`, domain.ToUUID(custID))
	require.NoError(t, err)

	composite := billingadmin.NewCompositeReadService(pool, &config.Config{})
	require.NotNil(t, composite)

	exportDir := t.TempDir()
	runner := billingadmin.NewJobRunner(composite, exportDir)

	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	jobID, err := runner.CreateJob(ctx, billingadmin.JobSpec{
		CustomerID: custID.String(),
		From:       from,
		To:         to,
		Format:     "csv",
	})
	require.NoError(t, err)
	require.NotEmpty(t, jobID)

	var status billingadmin.JobStatusDTO
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var ok bool
		status, ok = runner.GetJob(jobID)
		require.True(t, ok)
		switch status.Status {
		case billingadmin.JobStatusCompleted:
			goto done
		case billingadmin.JobStatusFailed:
			t.Fatalf("export job failed: %s", status.Error)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timed out waiting for billing export job")
done:
	require.Greater(t, status.Bytes, int64(len("id,amount_micro,ledger_type,created_at\n")),
		"export must include at least one ledger data row, not header only")

	f, _, err := runner.OpenDownload(jobID)
	require.NoError(t, err)
	defer f.Close()

	body, err := io.ReadAll(f)
	require.NoError(t, err)
	text := string(body)
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	require.GreaterOrEqual(t, len(lines), 2, "CSV must have header + >=1 ledger row")
	require.True(t, strings.HasPrefix(text, "id,amount_micro,ledger_type,created_at"))
	require.Contains(t, text, "TOPUP")
	require.Contains(t, text, "1500000")
	require.Equal(t, int64(len(body)), status.Bytes)

	testutil.LogFaultProof(t, "billing_export_ledger_nonzero", map[string]string{
		"bytes":  fmt.Sprintf("%d", status.Bytes),
		"format": "csv",
		"status": "completed",
	})
}
