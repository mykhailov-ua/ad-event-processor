package adminapi

import (
	"errors"
	"net/http"

	"espx/pkg/httpresponse"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrForbidden = errors.New("forbidden")

func WriteBillingError(w http.ResponseWriter, err error) {
	if st, ok := status.FromError(err); ok && st.Code() == codes.FailedPrecondition {
		httpresponse.Error(w, http.StatusConflict, "LEDGER_DRIFT", st.Message())
		return
	}
	httpresponse.WriteGRPCError(w, err)
}
