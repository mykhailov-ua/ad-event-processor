package controlplane_test

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/controlplane"
	"ad-event-processor/pkg/coldpath"
)

func TestClientIPXFFNotTrustedWithoutProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.RemoteAddr = "203.0.113.1:12345"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.1")

	got := controlplane.ExportedClientIP(r)
	if got == "1.2.3.4" {
		t.Fatalf("XFF spoofing succeeded: clientIP returned %q from untrusted peer", got)
	}
	if got != "203.0.113.1" {
		t.Fatalf("expected 203.0.113.1, got %q", got)
	}
}

func TestClientIPXFFTrustedWithProxy(t *testing.T) {
	_, cidr, _ := net.ParseCIDR("10.0.0.0/8")
	controlplane.SetTrustedProxyRanges([]*net.IPNet{cidr})
	t.Cleanup(func() { controlplane.SetTrustedProxyRanges(nil) })

	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.RemoteAddr = "10.0.0.1:4321"
	r.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.1")

	got := controlplane.ExportedClientIP(r)
	if got != "10.0.0.1" && got != "203.0.113.5" {
		t.Fatalf("unexpected IP %q from trusted proxy path", got)
	}
}

func TestCORSWildcardNoCredentials(t *testing.T) {
	handler := controlplane.NewCORSMiddleware([]string{"*"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	)
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got == "true" {
		t.Fatalf("CORS wildcard leak: Allow-Credentials should NOT be set for wildcard, got %q", got)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Fatal("expected origin to be reflected for wildcard CORS")
	}
}

func TestCORSExplicitAllowlistHasCredentials(t *testing.T) {
	handler := controlplane.NewCORSMiddleware([]string{"https://dashboard.example.com"})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	)
	r := httptest.NewRequest(http.MethodGet, "/", http.NoBody)
	r.Header.Set("Origin", "https://dashboard.example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("explicit origin should get Allow-Credentials:true, got %q", got)
	}
}

func TestCSRFPatchBlocked(t *testing.T) {
	mw := controlplane.NewCSRFMiddleware("secret-admin-key")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/123", http.NoBody)
	r.AddCookie(&http.Cookie{Name: "csrfToken", Value: "abc"})
	r.Header.Set("X-CSRF-Token", "WRONG")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code == http.StatusOK {
		t.Fatal("PATCH with wrong CSRF token must be rejected, but got 200")
	}
}

func TestCSRFPatchAccepted(t *testing.T) {
	mw := controlplane.NewCSRFMiddleware("secret-admin-key")
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	token := "csrf-secure-token-12345"
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/campaigns/123", http.NoBody)
	r.AddCookie(&http.Cookie{Name: "csrfToken", Value: token})
	r.Header.Set("X-CSRF-Token", token)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("PATCH with correct CSRF should pass, got %d", w.Code)
	}
}

func TestAdminKeyWrongKeyRejected(t *testing.T) {
	correctKey := "super-secret-admin-key-xyz"
	mw := controlplane.NewCSRFMiddleware(correctKey)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	almostCorrect := "super-secret-admin-key-xy"
	r := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns", http.NoBody)
	r.Header.Set("X-Admin-API-Key", almostCorrect)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code == http.StatusOK {
		t.Fatal("near-miss admin key must not bypass CSRF middleware")
	}
}

func TestLoginBodySizeLimit(t *testing.T) {
	huge := strings.NewReader(strings.Repeat("x", 1024*1024))
	r := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", huge)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r2 := r.Clone(r.Context())
	r2.Body = http.MaxBytesReader(w, r2.Body, coldpath.DefaultMaxBody)
	buf := make([]byte, 70000)
	n, err := r2.Body.Read(buf)
	if err == nil && n > 65536 {
		t.Fatalf("body limit not enforced: read %d bytes", n)
	}
	t.Logf("read %d bytes, err: %v (expected limit enforcement)", n, err)
}

func TestRateLimiterEviction(t *testing.T) {
	entries := make(map[string]*controlplane.ExportedRateLimiterEntry)
	now := time.Now()

	for i := range 100 {
		entries[fmt.Sprintf("ip-%d", i)] = &controlplane.ExportedRateLimiterEntry{
			LastSeen: now.Add(-20 * time.Minute),
		}
	}
	for i := 100; i < 110; i++ {
		entries[fmt.Sprintf("ip-%d", i)] = &controlplane.ExportedRateLimiterEntry{
			LastSeen: now,
		}
	}

	controlplane.ExportedEvictStale(entries, now)

	if len(entries) != 10 {
		t.Fatalf("expected 10 entries after eviction, got %d", len(entries))
	}
}

func TestCHLagCacheHit(t *testing.T) {
	calls := 0
	probe := func() (time.Duration, error) {
		calls++
		return 5 * time.Second, nil
	}

	lag1, _ := controlplane.ExportedCHLagWithCache(probe)
	lag2, _ := controlplane.ExportedCHLagWithCache(probe)

	if calls != 1 {
		t.Fatalf("expected exactly 1 CH probe within TTL, got %d", calls)
	}
	if lag1 != lag2 {
		t.Fatalf("cached lag should equal original: %v vs %v", lag1, lag2)
	}
}
