package coldpath

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

const DefaultMaxBody = 65536

func ParsePathUUID(r *http.Request, param string) (uuid.UUID, error) {
	if r == nil {
		return uuid.Nil, ErrNilRequest
	}
	return ParseUUID(r.PathValue(param))
}

var ErrNilRequest = fmt.Errorf("coldpath: nil request")

func DecodeRequestOrBadRequest[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (T, bool) {
	var zero T
	body, err := ReadLimitedBody(w, r, maxBytes)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return zero, false
	}
	v, err := DecodeBody[T](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return zero, false
	}
	return v, true
}

func ReadLimitedBody(w http.ResponseWriter, r *http.Request, maxBytes int64) ([]byte, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	return io.ReadAll(r.Body)
}

func DecodeBody[T any](body []byte) (T, error) {
	var v T
	err := json.Unmarshal(body, &v)
	return v, err
}

func DecodeRequest[T any](w http.ResponseWriter, r *http.Request, maxBytes int64) (T, error) {
	body, err := ReadLimitedBody(w, r, maxBytes)
	if err != nil {
		var zero T
		return zero, err
	}
	return DecodeBody[T](body)
}

func WritePaginatedJSON[T any](w http.ResponseWriter, items []T, total int64) {
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, items)
}
