package track

import (
	"net/http"
	"strings"
)

type CORS struct {
	allowAll bool
	origins  map[string]struct{}
}

func NewCORS(allowed []string) CORS {
	c := CORS{origins: make(map[string]struct{})}
	for _, raw := range allowed {
		o := strings.TrimSpace(raw)
		if o == "" {
			continue
		}
		if o == "*" {
			c.allowAll = true
			continue
		}
		c.origins[o] = struct{}{}
	}
	return c
}

func (c CORS) Match(origin string) bool {
	if origin == "" {
		return false
	}
	if c.allowAll {
		return true
	}
	_, ok := c.origins[origin]
	return ok
}

const (
	CORSHeaderBlock = "Access-Control-Allow-Origin: "
	CORSMethods     = "Access-Control-Allow-Methods: POST, OPTIONS\r\n"
	CORSHeaders     = "Access-Control-Allow-Headers: Content-Type\r\n"
	CORSVary        = "Vary: Origin\r\n"
	CORSPreflightOK = "HTTP/1.1 204 No Content\r\n" +
		CORSMethods +
		CORSHeaders +
		CORSVary +
		"Content-Length: 0\r\n" +
		"Connection: keep-alive\r\n\r\n"
)

func AppendCORSHeaders(dst []byte, origin string, cors CORS) []byte {
	if !cors.Match(origin) {
		return dst
	}
	dst = append(dst, CORSHeaderBlock...)
	dst = append(dst, origin...)
	dst = append(dst, '\r', '\n')
	dst = append(dst, CORSMethods...)
	dst = append(dst, CORSHeaders...)
	dst = append(dst, CORSVary...)
	return dst
}

func BuildCORSPreflight(origin string, cors CORS) []byte {
	if !cors.Match(origin) {
		return nil
	}
	dst := make([]byte, 0, len(CORSPreflightOK)+len(origin)+32)
	dst = append(dst, "HTTP/1.1 204 No Content\r\n"...)
	dst = append(dst, CORSHeaderBlock...)
	dst = append(dst, origin...)
	dst = append(dst, '\r', '\n')
	dst = append(dst, CORSMethods...)
	dst = append(dst, CORSHeaders...)
	dst = append(dst, CORSVary...)
	dst = append(dst, "Content-Length: 0\r\nConnection: keep-alive\r\n\r\n"...)
	return dst
}

func GnetTrackAcceptedHeaderBudget(origin string, cors CORS, bodyLen int, protobuf bool) int {
	n := len("HTTP/1.1 202 Accepted\r\n")
	if cors.Match(origin) {
		n += len(CORSHeaderBlock) + len(origin) + 2
		n += len(CORSMethods) + len(CORSHeaders) + len(CORSVary)
	}
	if protobuf {
		n += len("Content-Type: application/x-protobuf\r\nContent-Length: ")
	} else {
		n += len("Content-Type: application/json\r\nContent-Length: ")
	}
	n += len(strconvItoa(bodyLen))
	n += len("\r\nConnection: keep-alive\r\n\r\n")
	return n
}

func strconvItoa(n int) string {
	return string(appendInt(nil, int64(n)))
}

func appendInt(dst []byte, n int64) []byte {
	if n == 0 {
		return append(dst, '0')
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return append(dst, buf[i:]...)
}

func ApplyHTTPCORSHeaders(w http.ResponseWriter, origin string, cors CORS) {
	if !cors.Match(origin) {
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Add("Vary", "Origin")
}

func ServeHTTPCORSPreflight(w http.ResponseWriter, r *http.Request, cors CORS) {
	origin := r.Header.Get("Origin")
	if !cors.Match(origin) {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ApplyHTTPCORSHeaders(w, origin, cors)
	w.WriteHeader(http.StatusNoContent)
}
