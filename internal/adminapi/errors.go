package adminapi

import (
	"errors"
	"net/http"

	"espx/pkg/httpresponse"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrForbidden = errors.New("forbidden")

func WriteBillingGRPCError(w http.ResponseWriter, err error) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", st.Message())
			return
		case codes.NotFound:
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", st.Message())
			return
		case codes.FailedPrecondition:
			httpresponse.Error(w, http.StatusConflict, "LEDGER_DRIFT", st.Message())
			return
		}
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "billing request failed")
}
