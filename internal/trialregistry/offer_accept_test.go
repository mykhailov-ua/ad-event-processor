package trialregistry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func testEnqueueInput(telegramID string) EnqueuePendingInput {
	return EnqueuePendingInput{
		TelegramID:      telegramID,
		OfferVersion:    CurrentOfferVersion(),
		OfferAcceptedAt: time.Now().UTC(),
		AcceptSource:    "test",
	}
}

func testEnqueueInputUser(telegramID, username string) EnqueuePendingInput {
	in := testEnqueueInput(telegramID)
	in.TelegramUsername = username
	return in
}

func TestEnqueuePending_withoutOfferAcceptance_holdout(t *testing.T) {
	reg := New(t.TempDir()+"/trial.json", 0)
	_, err := reg.EnqueuePending(EnqueuePendingInput{TelegramID: "555"})
	require.ErrorIs(t, err, ErrOfferNotAccepted)
}

func TestAcceptOffer_allowsEnqueue(t *testing.T) {
	reg := New(t.TempDir()+"/trial.json", 0)
	require.NoError(t, reg.AcceptOffer("777", CurrentOfferVersion(), AcceptSourceTelegram))
	req, err := reg.EnqueuePending(EnqueuePendingInput{
		TelegramID:   "777",
		OfferVersion: CurrentOfferVersion(),
	})
	require.NoError(t, err)
	require.Equal(t, CurrentOfferVersion(), req.OfferVersion)
	require.False(t, req.OfferAcceptedAt.IsZero())
}
