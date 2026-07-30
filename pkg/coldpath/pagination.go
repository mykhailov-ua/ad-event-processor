package coldpath

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

type Page struct {
	Offset int
	Limit  int
}

func Paginate(cursor string, limit, maxLimit int) (Page, error) {
	if maxLimit <= 0 {
		maxLimit = 1000
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	offset, err := DecodeCursor(cursor)
	if err != nil {
		return Page{}, err
	}
	if offset < 0 {
		return Page{}, fmt.Errorf("coldpath: negative cursor offset")
	}
	return Page{Offset: offset, Limit: limit}, nil
}

func EncodeCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func DecodeCursor(cursor string) (int, error) {
	if cursor == "" {
		return 0, nil
	}
	decoded, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("coldpath: invalid cursor")
	}
	offset, err := strconv.Atoi(string(decoded))
	if err != nil {
		return 0, fmt.Errorf("coldpath: invalid cursor")
	}
	return offset, nil
}

func ClampLimitOffset(limit, offset, defaultLimit, maxLimit int32) (int32, int32) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}
