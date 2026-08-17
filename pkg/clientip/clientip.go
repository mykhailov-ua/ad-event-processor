// Package clientip extracts client IP addresses from HTTP requests and proxy headers.
package clientip

import (
	"net"
	"net/http"
	"strings"
)

type Trusted struct {
	exact map[string]bool
	nets  []*net.IPNet
}

func ParseTrusted(entries []string) Trusted {
	t := Trusted{exact: make(map[string]bool)}
	for _, raw := range entries {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		if _, ipNet, err := net.ParseCIDR(p); err == nil {
			t.nets = append(t.nets, ipNet)
			continue
		}
		if ip := net.ParseIP(p); ip != nil {
			t.exact[ip.String()] = true
		}
	}
	return t
}

func (t Trusted) Contains(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if t.exact[ip.String()] {
		return true
	}
	for _, cidr := range t.nets {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

func peerHost(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

func publicClientIP(ipStr string) (string, bool) {
	parsed := net.ParseIP(strings.TrimSpace(ipStr))
	if parsed == nil {
		return "", false
	}
	if parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsLinkLocalUnicast() {
		return "", false
	}
	return parsed.String(), true
}

func FromRequest(r *http.Request, trusted Trusted) string {
	if r == nil {
		return ""
	}
	remoteIP := peerHost(r.RemoteAddr)
	peerIP := net.ParseIP(remoteIP)
	if !trusted.Contains(peerIP) {
		return remoteIP
	}

	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		if ip, ok := publicClientIP(xri); ok {
			return ip
		}
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if ip, ok := lastPublicXFF(xff); ok {
			return ip
		}
	}
	return remoteIP
}

func FromProxyPeer(peerIP string, xff, xRealIP string, trusted Trusted) string {
	parsedPeer := net.ParseIP(peerIP)
	if !trusted.Contains(parsedPeer) {
		return peerIP
	}
	if xRealIP != "" {
		if ip, ok := publicClientIP(xRealIP); ok {
			return ip
		}
	}
	if xff != "" {
		if ip, ok := lastPublicXFF(xff); ok {
			return ip
		}
	}
	return peerIP
}

func lastPublicXFF(xff string) (string, bool) {
	last := len(xff)
	for i := len(xff) - 1; i >= -1; i-- {
		if i == -1 || xff[i] == ',' {
			start := i + 1
			for start < last && xff[start] == ' ' {
				start++
			}
			end := last
			for end > start && xff[end-1] == ' ' {
				end--
			}
			if start < end {
				if ip, ok := publicClientIP(xff[start:end]); ok {
					return ip, true
				}
			}
			last = i
		}
	}
	return "", false
}
