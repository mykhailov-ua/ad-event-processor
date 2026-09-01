package coldpath

import (
	"strconv"
	"strings"
)

// FormatMicroDisplay formats micro-unit integers for admin JSON tables.
// Uses ASCII thousands grouping only (no locale-specific separators).
func FormatMicroDisplay(m int64) string {
	return formatGroupedInt(m)
}

// FormatCountDisplay formats integer counts for admin JSON tables.
func FormatCountDisplay(n int64) string {
	return formatGroupedInt(n)
}

func formatGroupedInt(n int64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return sign + s
	}
	var b strings.Builder
	b.Grow(len(s) + len(s)/3)
	if sign != "" {
		b.WriteString(sign)
	}
	prefix := len(s) % 3
	if prefix == 0 {
		prefix = 3
	}
	b.WriteString(s[:prefix])
	for i := prefix; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}
