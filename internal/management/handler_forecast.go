package management

import (
	"errors"
	"net/http"
	"strconv"

	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

func writeForecastError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrForecastClickHouseTimeout) || errors.Is(err, ErrForecastUnavailable) {
		w.Header().Set("Retry-After", strconv.Itoa(ForecastRetryAfterSec()))
		httpresponse.JSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]string{
				"code":    "FORECAST_UNAVAILABLE",
				"message": err.Error(),
			},
			"retry_after": ForecastRetryAfterSec(),
		})
		return
	}
	if errors.Is(err, ErrClickHouseNotConfigured) {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	writeServiceError(w, err)
}

func (h *Handler) resolveForecastCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (*uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return nil, errForbidden
	}
	if u.IsUser() {
		if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil && *bodyCustomerID != u.CustomerID {
			return nil, errForbidden
		}
		cid := u.CustomerID
		return &cid, nil
	}
	if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil {
		return bodyCustomerID, nil
	}
	return nil, nil
}
