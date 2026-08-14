package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFromRequest_ignoresXFFFromUntrustedPeer(t *testing.T) {
	r := httptest.NewRequest("GET", "/", http.NoBody)
	r.RemoteAddr = "203.0.113.5:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.1")

	got := FromRequest(r, Trusted{})
	if got != "203.0.113.5" {
		t.Fatalf("expected peer IP, got %q", got)
	}
}

func TestFromRequest_trustsXFFFromTrustedPeer(t *testing.T) {
	trusted := ParseTrusted([]string{"10.0.0.1"})
	r := httptest.NewRequest("GET", "/", http.NoBody)
	r.RemoteAddr = "10.0.0.1:443"
	r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.2")

	got := FromRequest(r, trusted)
	if got != "198.51.100.2" {
		t.Fatalf("expected rightmost public XFF hop, got %q", got)
	}
}
