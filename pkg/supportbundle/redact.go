package supportbundle

import (
	"regexp"
	"strings"
)

var (
	urlPattern       = regexp.MustCompile(`(?i)https?://[^\s"'<>]+`)
	ipv4Pattern      = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	ipv6Pattern      = regexp.MustCompile(`\b(?:[0-9a-f]{1,4}:){4,7}[0-9a-f]{1,4}\b|(?:[0-9a-f]{1,4}:)*::(?:[0-9a-f]{1,4}:)*[0-9a-f]{1,4}\b`)
	stripeKeyPattern = regexp.MustCompile(`\bsk_(live|test)_[0-9a-zA-Z]+\b`)
	hexSaltPattern   = regexp.MustCompile(`(?i)pii_salt_hex=[0-9a-f]+`)
	jsonSecretKeys   = []string{"license_key", "target_url", "creative", "client_secret", "api_key", "ip", "ip_address"}
	kvSecretKeys     = []string{"license_key", "target_url", "creative", "client_secret", "api_key"}
)

func RedactLine(line string) string {
	s := line
	for _, key := range jsonSecretKeys {
		re := regexp.MustCompile(`(?i)"` + key + `"\s*:\s*"[^"]*"`)
		s = re.ReplaceAllString(s, `"`+key+`":"***"`)
	}
	for _, key := range kvSecretKeys {
		re := regexp.MustCompile(`(?i)` + key + `=[^\s,]+`)
		s = re.ReplaceAllString(s, key+"=***")
	}
	s = urlPattern.ReplaceAllString(s, "[REDACTED_URL]")
	s = stripeKeyPattern.ReplaceAllString(s, "[REDACTED_SECRET]")
	s = ipv4Pattern.ReplaceAllString(s, "[REDACTED_IP]")
	s = ipv6Pattern.ReplaceAllString(s, "[REDACTED_IP]")
	s = hexSaltPattern.ReplaceAllString(s, "pii_salt_hex=***")
	rePII := regexp.MustCompile(`(?i)"pii_salt_hex"\s*:\s*"[^"]*"`)
	s = rePII.ReplaceAllString(s, `"pii_salt_hex":"***"`)
	return s
}

func RedactLog(lines []string) []byte {
	if len(lines) == 0 {
		return []byte("\n")
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = RedactLine(line)
	}
	return []byte(strings.Join(out, "\n") + "\n")
}
