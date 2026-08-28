package controlplane

import (
	"net/http"
	"net/url"
	"strings"
)

type ListSortDTO struct {
	Field string `json:"field"`
	Order string `json:"order"`
}

type ListEnvelope[T any] struct {
	Items          []T               `json:"items"`
	Total          int64             `json:"total"`
	Limit          int32             `json:"limit"`
	Offset         int32             `json:"offset"`
	Freshness      DataFreshnessDTO  `json:"freshness,omitempty"`
	FiltersApplied map[string]string `json:"filters_applied,omitempty"`
	Sort           *ListSortDTO      `json:"sort,omitempty"`
}

func parseListSort(r *http.Request, allowed map[string]struct{}, defaultField string) (field, order string, err error) {
	field = strings.TrimSpace(r.URL.Query().Get("sort"))
	if field == "" {
		field = defaultField
	}
	if _, ok := allowed[field]; !ok {
		return "", "", invalidQueryError("invalid sort")
	}
	order = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return "", "", invalidQueryError("invalid order")
	}
	return field, order, nil
}

func filtersAppliedFromQuery(r *http.Request, keys ...string) map[string]string {
	out := make(map[string]string, len(keys))
	q := r.URL.Query()
	for _, key := range keys {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			out[key] = v
		}
	}
	return out
}

func mergeQueryFilters(filters map[string]string) string {
	if len(filters) == 0 {
		return ""
	}
	vals := url.Values{}
	for k, v := range filters {
		vals.Set(k, v)
	}
	return vals.Encode()
}
