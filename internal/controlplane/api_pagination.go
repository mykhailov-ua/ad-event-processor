package controlplane

import (
	"net/http"
	"strconv"

	"espx/pkg/coldpath"
)

type invalidQueryError string

func errInvalidQuery(msg string) error {
	return invalidQueryError(msg)
}

func (e invalidQueryError) Error() string { return string(e) }

func parseAPIPagination(r *http.Request) (int32, int32) {
	limit := int32(50)
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}
	offset := int32(0)
	if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32); err == nil && o > 0 {
		offset = int32(o)
	}
	return coldpath.ClampLimitOffset(limit, offset, 50, 1000)
}
