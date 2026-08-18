package coldpath

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const (
	DefaultMaxBody = 65536

	SelfServePaymentIntentMaxBody = 16 * 1024

	PaymentWebhookMaxBody = DefaultMaxBody

	AlertmanagerWebhookMaxBody = 1 << 20

	RegionIngestMaxBody = 4 * 1024 * 1024
)

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

func CloseHTTPResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}

func ParseAPIPagination(r *http.Request) (int32, int32) {
	return ParseAPIPaginationWith(r, 50, 1000)
}

func ParseAPIPaginationWith(r *http.Request, defaultLimit, maxLimit int32) (int32, int32) {
	limit := defaultLimit
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}
	offset := int32(0)
	if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32); err == nil && o > 0 {
		offset = int32(o)
	}
	return ClampLimitOffset(limit, offset, defaultLimit, maxLimit)
}
