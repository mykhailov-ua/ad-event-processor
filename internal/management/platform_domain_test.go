package management

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"espx/internal/ingestion"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatform_mapServiceError_forbidden(t *testing.T) {
	t.Parallel()
	status, code, _ := mapServiceError(errForbidden)
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, "FORBIDDEN", code)
}

func TestPlatform_mapServiceError_conflict(t *testing.T) {
	t.Parallel()
	status, code, msg := mapServiceError(ErrSlotMigrationNotReady)
	assert.Equal(t, http.StatusConflict, status)
	assert.Equal(t, "CONFLICT", code)
	assert.Contains(t, msg, "migration")
}

func TestPlatform_mapServiceError_limitExceeded(t *testing.T) {
	t.Parallel()
	status, code, _ := mapServiceError(ErrSelfServeActiveCampaignLimit)
	assert.Equal(t, http.StatusTooManyRequests, status)
	assert.Equal(t, "LIMIT_EXCEEDED", code)
}

func TestPlatform_mapNotFound(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, mapNotFound(pgx.ErrNoRows, ErrCampaignNotFound), ErrCampaignNotFound)
	assert.NoError(t, mapNotFound(nil, ErrCampaignNotFound))
	assert.Equal(t, errors.New("other"), mapNotFound(errors.New("other"), ErrCampaignNotFound))
}

func TestPlatform_isNotFoundError_brand(t *testing.T) {
	t.Parallel()
	assert.True(t, isNotFoundError(ErrBrandNotFound))
}

func TestPlatform_badRequestMessage_insufficientBalance(t *testing.T) {
	t.Parallel()
	msg, ok := badRequestMessage(ErrInsufficientBalance)
	assert.True(t, ok)
	assert.Contains(t, msg, "insufficient")
}

func TestPlatform_conflictMessage_default(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "conflict", conflictMessage(errors.New("other")))
}

func TestPlatform_writeServiceError_internal(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeServiceError(rec, errors.New("db exploded"))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPlatform_badRequestMessage_refund(t *testing.T) {
	t.Parallel()
	msg, ok := badRequestMessage(ErrRefundExceedsTopup)
	assert.True(t, ok)
	assert.Contains(t, msg, "refund")
}

func TestPlatform_conflictMessage_slotActive(t *testing.T) {
	t.Parallel()
	assert.Contains(t, conflictMessage(ingestion.ErrSlotMapAlreadyActive), "active")
}

func TestPlatform_mapServiceError_nil(t *testing.T) {
	t.Parallel()
	status, code, msg := mapServiceError(nil)
	assert.Equal(t, http.StatusOK, status)
	assert.Empty(t, code)
	assert.Empty(t, msg)
}

func TestPlatform_badRequestMessage_pacing(t *testing.T) {
	t.Parallel()
	msg, ok := badRequestMessage(ErrInvalidPacingMode)
	assert.True(t, ok)
	assert.Contains(t, msg, "pacing")
}

func TestPlatform_writeServiceError_clientError(t *testing.T) {
	t.Parallel()
	rec := httptest.NewRecorder()
	writeServiceError(rec, ErrCustomerNotFound)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestPlatform_mapServiceError_invalidQuery(t *testing.T) {
	t.Parallel()
	status, code, msg := mapServiceError(invalidQueryError("bad filter"))
	assert.Equal(t, http.StatusBadRequest, status)
	assert.Equal(t, "BAD_REQUEST", code)
	assert.Equal(t, "bad filter", msg)
}

func TestPlatform_mapServiceError_supplyInvalid(t *testing.T) {
	t.Parallel()
	status, code, _ := mapServiceError(ErrSellersJSONInvalid)
	assert.Equal(t, http.StatusServiceUnavailable, status)
	assert.Equal(t, "SUPPLY_INVALID", code)
}

func TestPlatform_errValidation(t *testing.T) {
	t.Parallel()
	err := errValidation("field required")
	var ve validationError
	require.ErrorAs(t, err, &ve)
}

func TestPlatform_DomainMapped(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "platform", FileDomain("handler_errors.go"))
}
