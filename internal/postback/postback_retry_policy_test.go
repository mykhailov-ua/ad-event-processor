package postback

import (
	"context"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type statusSequenceAdapter struct {
	codes []int
	calls atomic.Int32
}

func (a *statusSequenceAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate, token string) error {
	n := int(a.calls.Add(1)) - 1
	code := a.codes[len(a.codes)-1]
	if n < len(a.codes) {
		code = a.codes[n]
	}
	if code >= 200 && code < 300 {
		return nil
	}
	return &DispatchHTTPError{StatusCode: code, Body: "mock"}
}

func TestCAPI_RetryPolicy_4xxNoRetry(t *testing.T) {
	adapter := &statusSequenceAdapter{codes: []int{http.StatusBadRequest}}
	worker := NewPostbackWorker(nil, []byte("postback-encryption-secret-key32"))
	err := worker.dispatchWithRetry(context.Background(), adapter, &PostbackPayload{
		CampaignID: uuid.New(),
		ClickID:    "c1",
	}, "https://example.com", "tok", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "permanent client error")
	require.Equal(t, int32(1), adapter.calls.Load())
}

func TestCAPI_RetryPolicy_5xxBackoffMax5(t *testing.T) {
	adapter := &statusSequenceAdapter{codes: []int{
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusInternalServerError,
	}}
	worker := NewPostbackWorker(nil, []byte("postback-encryption-secret-key32"))
	err := worker.dispatchWithRetry(context.Background(), adapter, &PostbackPayload{
		CampaignID: uuid.New(),
		ClickID:    "c1",
	}, "https://example.com", "tok", nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed after 5 attempts")
	require.Equal(t, int32(5), adapter.calls.Load())
}

func TestCAPI_RetryPolicy_5xxSucceedsOnRetry(t *testing.T) {
	adapter := &statusSequenceAdapter{codes: []int{http.StatusServiceUnavailable, http.StatusOK}}
	worker := NewPostbackWorker(nil, []byte("postback-encryption-secret-key32"))
	err := worker.dispatchWithRetry(context.Background(), adapter, &PostbackPayload{
		CampaignID: uuid.New(),
		ClickID:    "c1",
	}, "https://example.com", "tok", nil)
	require.NoError(t, err)
	require.Equal(t, int32(2), adapter.calls.Load())
}

func TestDispatchHTTPError_Permanent(t *testing.T) {
	require.True(t, (&DispatchHTTPError{StatusCode: 400}).Permanent())
	require.True(t, (&DispatchHTTPError{StatusCode: 429}).Permanent())
	require.False(t, (&DispatchHTTPError{StatusCode: 500}).Permanent())
	require.False(t, errors.Is(&DispatchHTTPError{StatusCode: 400}, &DispatchHTTPError{}))
}
