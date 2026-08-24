package controlplane

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatform_errorMapping(t *testing.T) {
	t.Parallel()

	t.Run("mapServiceError", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			err        error
			wantStatus int
			wantCode   string
			wantMsg    string
		}{
			{nil, http.StatusOK, "", ""},
			{errForbidden, http.StatusForbidden, "FORBIDDEN", ""},
			{ErrSlotMigrationNotReady, http.StatusConflict, "CONFLICT", "migration"},
			{ErrSelfServeActiveCampaignLimit, http.StatusTooManyRequests, "LIMIT_EXCEEDED", ""},
			{invalidQueryError("bad filter"), http.StatusBadRequest, "BAD_REQUEST", "bad filter"},
			{ErrSellersJSONInvalid, http.StatusServiceUnavailable, "SUPPLY_INVALID", ""},
		}
		for _, tc := range cases {
			status, code, msg := mapServiceError(tc.err)
			assert.Equal(t, tc.wantStatus, status, tc.err)
			assert.Equal(t, tc.wantCode, code, tc.err)
			if tc.wantMsg != "" {
				assert.Contains(t, msg, tc.wantMsg, tc.err)
			}
		}
	})

	t.Run("mapNotFound", func(t *testing.T) {
		t.Parallel()
		assert.ErrorIs(t, mapNotFound(pgx.ErrNoRows, ErrCampaignNotFound), ErrCampaignNotFound)
		assert.NoError(t, mapNotFound(nil, ErrCampaignNotFound))
		assert.Equal(t, errors.New("other"), mapNotFound(errors.New("other"), ErrCampaignNotFound))
	})

	t.Run("isNotFoundError", func(t *testing.T) {
		t.Parallel()
		assert.True(t, isNotFoundError(ErrBrandNotFound))
	})

	t.Run("badRequestMessage", func(t *testing.T) {
		t.Parallel()
		cases := []struct {
			err     error
			wantOK  bool
			contain string
		}{
			{ErrInsufficientBalance, true, "insufficient"},
			{ErrRefundExceedsTopup, true, "refund"},
			{ErrInvalidPacingMode, true, "pacing"},
		}
		for _, tc := range cases {
			msg, ok := badRequestMessage(tc.err)
			assert.Equal(t, tc.wantOK, ok, tc.err)
			if tc.contain != "" {
				assert.Contains(t, msg, tc.contain, tc.err)
			}
		}
	})

	t.Run("conflictMessage", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "conflict", conflictMessage(errors.New("other")))
		assert.Contains(t, conflictMessage(domain.ErrSlotMapAlreadyActive), "active")
	})

	t.Run("writeServiceError", func(t *testing.T) {
		t.Parallel()
		rec := httptest.NewRecorder()
		writeServiceError(rec, errors.New("db exploded"))
		assert.Equal(t, http.StatusInternalServerError, rec.Code)

		rec = httptest.NewRecorder()
		writeServiceError(rec, ErrCustomerNotFound)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestPlatform_errValidation(t *testing.T) {
	t.Parallel()
	err := errValidation("field required")
	var ve validationError
	require.ErrorAs(t, err, &ve)
}
