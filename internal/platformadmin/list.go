package platformadmin

import (
	"errors"
	"net/http"
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
	FiltersApplied map[string]string `json:"filters_applied,omitempty"`
	Sort           *ListSortDTO      `json:"sort,omitempty"`
}

func parseListSort(r *http.Request, allowed map[string]struct{}, defaultField string) (field, order string, err error) {
	field = strings.TrimSpace(r.URL.Query().Get("sort"))
	if field == "" {
		field = defaultField
	}
	if _, ok := allowed[field]; !ok {
		return "", "", errors.New("invalid sort")
	}
	order = strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return "", "", errors.New("invalid order")
	}
	return field, order, nil
}
