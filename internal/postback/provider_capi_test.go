package postback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestResolveGoogleConversionAction(t *testing.T) {
	if got := resolveGoogleConversionAction("customers/1/conversionActions/9", "conversion"); got != "customers/1/conversionActions/9" {
		t.Fatalf("got %q", got)
	}
	if got := resolveGoogleConversionAction("https://example.com/upload", "lead"); got != "lead" {
		t.Fatalf("got %q", got)
	}
}

func TestGoogleAdapter_UsesConversionActionFromTemplate(t *testing.T) {
	var body GoogleCAPIPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &GoogleAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		GCLID:       "gclid-1",
		PayoutMicro: 5_000_000,
	}, srv.URL, "token")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(body.Conversions) != 1 {
		t.Fatalf("conversions: %+v", body.Conversions)
	}
}

func TestTikTokAdapter_PostsToCustomEndpoint(t *testing.T) {
	var body TikTokCAPIPayload
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		if r.Header.Get("Access-Token") != "tok" {
			t.Fatalf("token header %q", r.Header.Get("Access-Token"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &TikTokAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		CampaignID: uuid.New(),
		ClickID:    "clk",
		TTCLID:     "tt-1",
		EventType:  "conversion",
	}, srv.URL, "tok")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if len(body.Events) != 1 || body.Events[0].Context.Ttclid != "tt-1" {
		t.Fatalf("events %+v", body.Events)
	}
}

func TestTikTokPixelFromTemplate(t *testing.T) {
	if got := resolveTikTokPixelCode("PIXEL123"); got != "PIXEL123" {
		t.Fatalf("got %q", got)
	}
	if got := resolveTikTokPixelCode("https://example.com/track"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestFacebookAdapter_AcceptsCustomGraphURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.String(), "/12345/events") {
			t.Fatalf("url %s", r.URL.String())
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &FacebookAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		FBCLID:    "fbclick",
		EventType: "conversion",
	}, srv.URL+"/12345/events", "token")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
}
