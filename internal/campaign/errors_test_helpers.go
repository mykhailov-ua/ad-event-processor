package campaign

import (
	"errors"
	"net/http"
)

func mapServiceError(err error) (int, string, string) {
	if err == nil {
		return http.StatusInternalServerError, "INTERNAL", "internal error"
	}
	if errors.Is(err, ErrCampaignNotFound) {
		return http.StatusNotFound, "NOT_FOUND", err.Error()
	}
	if errors.Is(err, ErrForbidden) {
		return http.StatusForbidden, "FORBIDDEN", err.Error()
	}
	if errors.Is(err, ErrValidation) {
		return http.StatusBadRequest, "BAD_REQUEST", err.Error()
	}
	return http.StatusInternalServerError, "INTERNAL", err.Error()
}
