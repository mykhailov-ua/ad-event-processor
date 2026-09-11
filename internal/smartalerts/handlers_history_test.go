package smartalerts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type stubSmartAlertHistoryService struct {
	lastLimit  int
	lastOffset int
}

func (s *stubSmartAlertHistoryService) ListSmartAlertRules(context.Context, uuid.UUID) ([]RuleDTO, error) {
	return nil, nil
}

func (s *stubSmartAlertHistoryService) CreateSmartAlertRule(context.Context, UpsertRuleRequest) (RuleDTO, error) {
	return RuleDTO{}, nil
}

func (s *stubSmartAlertHistoryService) UpdateSmartAlertRule(context.Context, uuid.UUID, UpsertRuleRequest) (RuleDTO, error) {
	return RuleDTO{}, nil
}

func (s *stubSmartAlertHistoryService) DeleteSmartAlertRule(context.Context, uuid.UUID) error {
	return nil
}

func (s *stubSmartAlertHistoryService) ListSmartAlertHistory(_ context.Context, _ uuid.UUID, limit, offset int) ([]EventDTO, error) {
	s.lastLimit = limit
	s.lastOffset = offset
	return []EventDTO{}, nil
}

func (s *stubSmartAlertHistoryService) AckSmartAlertEvent(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func TestHTTPHandlers_listHistory_passesOffset_holdout(t *testing.T) {
	t.Parallel()
	customerID := uuid.New()
	stub := &stubSmartAlertHistoryService{}
	h := &HTTPHandlers{
		Service: stub,
		ApplyRateLimit: func(next http.HandlerFunc) http.HandlerFunc {
			return next
		},
		RequirePermission: func(_ string, next http.HandlerFunc) http.HandlerFunc {
			return next
		},
	}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/smart-alerts/history?customer_id="+customerID.String()+"&limit=25&offset=50",
		nil,
	)
	rec := httptest.NewRecorder()
	h.listHistory(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 25, stub.lastLimit)
	require.Equal(t, 50, stub.lastOffset)
}
