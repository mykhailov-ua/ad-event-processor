package controlplane

import (
	"ad-event-processor/internal/opsadmin"
)

type ListResponse[T any] struct {
	Items []T   `json:"items"`
	Total int64 `json:"total"`
}

type OffsetListResponse[T any] struct {
	Items  []T   `json:"items"`
	Total  int64 `json:"total"`
	Limit  int32 `json:"limit"`
	Offset int32 `json:"offset"`
}

type CursorListResponse[T any] struct {
	Items      []T    `json:"items"`
	Total      int64  `json:"total"`
	NextCursor string `json:"next_cursor"`
	Limit      int32  `json:"limit"`
}

type ItemsResponse[T any] struct {
	Items []T `json:"items"`
}

type PaymentHistoryRow = opsadmin.PaymentHistoryRow

type PaymentIntentListResponse = OffsetListResponse[PaymentHistoryRow]

type PaymentIntentCreatedResponse struct {
	IntentID       string `json:"intent_id"`
	Status         string `json:"status"`
	CheckoutURL    string `json:"checkout_url"`
	ProviderRef    string `json:"provider_ref"`
	DepositAddress string `json:"deposit_address,omitempty"`
	DepositNetwork string `json:"deposit_network,omitempty"`
	DepositQRSVG   string `json:"deposit_qr_svg,omitempty"`
}

type ExportJobCreatedResponse struct {
	JobID string `json:"job_id"`
}

type IDCreatedResponse struct {
	ID string `json:"id"`
}

type APIKeyCreatedResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	RawKey    string   `json:"raw_key"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

type DeliveryListResponse = ItemsResponse[DeliveryDTO]

type LedgerLinesListResponse = CursorListResponse[LedgerLineDTO]
