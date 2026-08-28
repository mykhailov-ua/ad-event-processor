package reports

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCustomerFraudByDimension_unknownDimension400(t *testing.T) {
	t.Parallel()
	h := &ReportsHTTPHandlers{}
	mux := http.NewServeMux()
	h.Register(mux)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/customer-fraud-by-dimension?customer_id="+uuid.New().String()+"&dimension=invalid", http.NoBody)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCustomerFraudByDimension_allowedDimensions(t *testing.T) {
	t.Parallel()
	for dim := range allowedFraudDimensions {
		_, ok := allowedFraudDimensions[dim]
		require.True(t, ok)
	}
	require.Len(t, allowedFraudDimensions, 5)
}
