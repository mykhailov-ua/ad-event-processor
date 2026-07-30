package coldpath

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"espx/pkg/httpresponse"
)

const DefaultMaxBody = 65536

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
