package httpresponse

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestWriteGRPCError(t *testing.T) {
	rec := httptest.NewRecorder()
	WriteGRPCError(rec, status.Error(codes.NotFound, "missing"))
	assert.Equal(t, http.StatusNotFound, rec.Code)

	rec = httptest.NewRecorder()
	WriteGRPCError(rec, status.Error(codes.InvalidArgument, "bad"))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
