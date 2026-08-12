package ingestion

import (
	"strings"
)

type trackCORS struct {
	allowAll bool
	origins  map[string]struct{}
}

func newTrackCORS(allowed []string) trackCORS {
	c := trackCORS{origins: make(map[string]struct{})}
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

func (c trackCORS) match(origin string) bool {
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
	trackCORSHeaderBlock = "Access-Control-Allow-Origin: "
	trackCORSMethods     = "Access-Control-Allow-Methods: POST, OPTIONS\r\n"
	trackCORSHeaders     = "Access-Control-Allow-Headers: Content-Type\r\n"
	trackCORSVary        = "Vary: Origin\r\n"
	trackCORSPreflightOK = "HTTP/1.1 204 No Content\r\n" +
		trackCORSMethods +
		trackCORSHeaders +
		trackCORSVary +
		"Content-Length: 0\r\n" +
		"Connection: keep-alive\r\n\r\n"
)

func appendTrackCORSHeaders(dst []byte, origin string, cors trackCORS) []byte {
	if !cors.match(origin) {
		return dst
	}
	dst = append(dst, trackCORSHeaderBlock...)
	dst = append(dst, origin...)
	dst = append(dst, '\r', '\n')
	dst = append(dst, trackCORSMethods...)
	dst = append(dst, trackCORSHeaders...)
	dst = append(dst, trackCORSVary...)
	return dst
}

func buildTrackCORSPreflight(origin string, cors trackCORS) []byte {
	if !cors.match(origin) {
		return nil
	}
	dst := make([]byte, 0, len(trackCORSPreflightOK)+len(origin)+32)
	dst = append(dst, "HTTP/1.1 204 No Content\r\n"...)
	dst = append(dst, trackCORSHeaderBlock...)
	dst = append(dst, origin...)
	dst = append(dst, '\r', '\n')
	dst = append(dst, trackCORSMethods...)
	dst = append(dst, trackCORSHeaders...)
	dst = append(dst, trackCORSVary...)
	dst = append(dst, "Content-Length: 0\r\nConnection: keep-alive\r\n\r\n"...)
	return dst
}
