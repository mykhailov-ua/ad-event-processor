package campaign

import (
	"encoding/json"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
)

func FormatOptionalText(t pgtype.Text) string {
	if !t.Valid {
		return ""
	}
	return strings.TrimSpace(t.String)
}

func ClickQueryParamsFromRaw(raw []byte) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var out map[string]string
	if err := json.Unmarshal(raw, &out); err != nil || len(out) == 0 {
		return nil
	}
	return normalizeClickQueryParams(out)
}
