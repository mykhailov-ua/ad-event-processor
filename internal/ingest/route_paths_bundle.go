package ingest

const (
	telegramPathClick      = "/tg/click"
	telegramPathImpression = "/tg/impression"
	safePageVerifyPath     = "/track/verify"

	safePageVerifyMinEvents  = 15
	safePageVerifyMaxBody    = 8192
	safePageVerifyRateLimit  = 30
	safePageVerifyRateWindow = 60
)

func appendInt64(dst []byte, v int64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	if v < 0 {
		dst = append(dst, '-')
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, buf[i:]...)
}
