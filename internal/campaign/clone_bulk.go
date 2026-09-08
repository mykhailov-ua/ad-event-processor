package campaign

import (
	"errors"
	"strings"

	"github.com/google/uuid"
)

const BulkCloneCampaignMaxSync = 50

type BulkCloneCampaignsRequest struct {
	SourceCampaignIDs []uuid.UUID
	CustomerID        uuid.UUID
	NamePrefix        string
	NameSuffix        string
	Options           CloneCampaignOptions
	IdempotencyKey    string
}

type BulkCloneCampaignResultRow struct {
	SourceID  string `json:"source_id"`
	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	OK        bool   `json:"ok"`
	ErrorCode string `json:"error_code,omitempty"`
}

type BulkCloneCampaignsResult struct {
	Results []BulkCloneCampaignResultRow `json:"results"`
}

func BulkCloneIdempotencyKey(bulkKey string, sourceID uuid.UUID) string {
	return strings.TrimSpace(bulkKey) + ":" + sourceID.String()
}

func BulkCloneCampaignErrorCode(err error) string {
	switch {
	case errors.Is(err, ErrValidation):
		return "validation_error"
	case errors.Is(err, ErrCampaignNotFound):
		return "not_found"
	case errors.Is(err, ErrCustomerNotFound):
		return "customer_mismatch"
	case errors.Is(err, ErrInsufficientBalance):
		return "insufficient_balance"
	case errors.Is(err, ErrForbidden):
		return "forbidden"
	default:
		return "error"
	}
}
