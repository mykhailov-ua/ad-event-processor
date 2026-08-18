package controlplane

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestJobRunner_CreateJob_invalidCustomer(t *testing.T) {
	t.Parallel()
	runner := NewJobRunner(&CompositeReadService{}, t.TempDir())
	_, err := runner.CreateJob(t.Context(), JobSpec{
		CustomerID: "not-a-uuid",
		From:       time.Now().UTC().Format(time.RFC3339),
		To:         time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
		Format:     "csv",
	})
	require.Error(t, err)
}

func TestJobSpec_formatValidation(t *testing.T) {
	t.Parallel()
	runner := NewJobRunner(&CompositeReadService{}, t.TempDir())
	_, err := runner.CreateJob(t.Context(), JobSpec{
		CustomerID: "550e8400-e29b-41d4-a716-446655440000",
		From:       "2026-06-01T00:00:00Z",
		To:         "2026-06-30T00:00:00Z",
		Format:     "xml",
	})
	require.Error(t, err)
}

type slowLedgerExportReader struct {
	delay time.Duration
}

func (s *slowLedgerExportReader) ListLedgerLinesInWindow(ctx context.Context, _ uuid.UUID, _, _ time.Time, _ int64, _ int32) ([]LedgerLineDTO, string, error) {
	timer := time.NewTimer(s.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return nil, "", ctx.Err()
	case <-timer.C:
		return nil, "1", nil
	}
}

func TestJobRunner_runJob_timesOut(t *testing.T) {
	t.Parallel()
	custID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	runner := NewJobRunner(&slowLedgerExportReader{delay: 100 * time.Millisecond}, t.TempDir())

	jobID := uuid.New().String()
	runner.mu.Lock()
	runner.jobs[jobID] = &jobRecord{
		spec:       JobSpec{Format: "csv"},
		customerID: custID,
		from:       time.Now().UTC().Add(-time.Hour),
		to:         time.Now().UTC(),
		status:     JobStatusPending,
		createdAt:  time.Now().UTC(),
	}
	runner.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	runner.runJob(ctx, jobID)

	status, ok := runner.GetJob(jobID)
	require.True(t, ok)
	require.Equal(t, JobStatusFailed, status.Status)
	require.Contains(t, status.Error, "deadline exceeded")
}
