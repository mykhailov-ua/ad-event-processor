package billingadmin

import (
	"errors"
	"net/http"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	CustomerBalanceDTO = campaign.CustomerBalanceDTO
	BalanceLedgerDTO   = campaign.BalanceLedgerDTO
	LedgerExportResult = campaign.LedgerExportResult
	UsageExportResult  = campaign.UsageExportResult
)

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

type LedgerLinesListResponse = CursorListResponse[LedgerLineDTO]
type DeliveryListResponse = ItemsResponse[DeliveryDTO]

var ErrForbidden = errors.New("forbidden")

type UsageExportSpec struct {
	CustomerID *uuid.UUID
	CostCenter string
	FromDate   time.Time
	ToDate     time.Time
	Cursor     UsageExportCursor
}

type UsageExportCursor struct {
	CustomerID uuid.UUID
	UsageDate  time.Time
	Meter      string
	Valid      bool
}

func errValidation(msg string) error {
	return errors.New(msg)
}

func WriteBillingError(w http.ResponseWriter, err error) {
	if st, ok := status.FromError(err); ok && st.Code() == codes.FailedPrecondition {
		httpresponse.Error(w, http.StatusConflict, "LEDGER_DRIFT", st.Message())
		return
	}
	httpresponse.WriteGRPCError(w, err)
}
