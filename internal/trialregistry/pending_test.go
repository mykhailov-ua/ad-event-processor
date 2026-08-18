package trialregistry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnqueuePending_idempotentOpen(t *testing.T) {
	reg := New(t.TempDir()+"/trial.json", 0)

	first, err := reg.EnqueuePending(EnqueuePendingInput{
		TelegramID:       "12345",
		TelegramUsername: "buyer",
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.ID)
	require.Equal(t, PendingStatusOpen, first.Status)

	second, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "12345"})
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
}

func TestEnqueuePending_deniesUsedTelegramAnchor(t *testing.T) {
	path := t.TempDir() + "/trial.json"
	reg := New(path, 0)
	dep := "550e8400-e29b-41d4-a716-446655440000"
	require.NoError(t, reg.RecordPilotIssue(RecordInput{
		TelegramID:   "777",
		DeploymentID: dep,
		LicenseKey:   dep,
		ValidUntil:   time.Now().UTC().Add(24 * time.Hour),
	}))

	_, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "777"})
	require.ErrorIs(t, err, ErrTrialTelegramUsed)
}

func TestPreparePendingIssue_assignsDeploymentID(t *testing.T) {
	reg := New(t.TempDir()+"/trial.json", 0)

	req, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "888"})
	require.NoError(t, err)

	approved, err := reg.PreparePendingIssue(req.ID, "")
	require.NoError(t, err)
	require.Equal(t, PendingStatusApproved, approved.Status)
	require.NotEmpty(t, approved.DeploymentID)
	require.Equal(t, "888", approved.TelegramID)

	_, err = reg.PreparePendingIssue(req.ID, "")
	require.ErrorIs(t, err, ErrPendingNotOpen)
}

func TestRejectPending(t *testing.T) {
	reg := New(t.TempDir()+"/trial.json", 0)

	req, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "999"})
	require.NoError(t, err)

	require.NoError(t, reg.RejectPending(req.ID, "spam"))
	_, err = reg.PreparePendingIssue(req.ID, "")
	require.ErrorIs(t, err, ErrPendingNotOpen)

	open, err := reg.ListPending()
	require.NoError(t, err)
	require.Empty(t, open)
}

func TestListPending_filtersClosed(t *testing.T) {
	reg := New(t.TempDir()+"/trial.json", 0)

	openReq, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "100"})
	require.NoError(t, err)
	closedReq, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "101"})
	require.NoError(t, err)
	require.NoError(t, reg.RejectPending(closedReq.ID, "test"))

	pending, err := reg.ListPending()
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, openReq.ID, pending[0].ID)
}
