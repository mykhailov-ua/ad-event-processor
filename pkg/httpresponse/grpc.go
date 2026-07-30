package httpresponse

import (
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func WriteGRPCError(w http.ResponseWriter, err error) {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			Error(w, http.StatusBadRequest, "BAD_REQUEST", st.Message())
			return
		case codes.NotFound:
			Error(w, http.StatusNotFound, "NOT_FOUND", st.Message())
			return
		case codes.AlreadyExists:
			Error(w, http.StatusConflict, "CONFLICT", st.Message())
			return
		case codes.FailedPrecondition:
			Error(w, http.StatusConflict, "CONFLICT", st.Message())
			return
		case codes.Unauthenticated:
			Error(w, http.StatusUnauthorized, "UNAUTHORIZED", st.Message())
			return
		case codes.PermissionDenied:
			Error(w, http.StatusForbidden, "FORBIDDEN", st.Message())
			return
		case codes.ResourceExhausted:
			Error(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS", st.Message())
			return
		case codes.Unavailable:
			Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", st.Message())
			return
		}
	}
	Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}
