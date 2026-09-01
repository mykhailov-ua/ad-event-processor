package coldpath

import (
	"strings"
	"time"
)

// RFC3339Display formats an RFC3339/RFC3339Nano timestamp for admin JSON tables.
// Unparseable input is returned unchanged so wire values stay visible.
func RFC3339Display(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	ts, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		ts, err = time.Parse(time.RFC3339, s)
		if err != nil {
			return raw
		}
	}
	return ts.UTC().Format("2006-01-02 15:04 UTC")
}
