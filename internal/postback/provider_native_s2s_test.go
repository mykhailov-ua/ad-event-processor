package postback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func TestTaboolaAdapter_S2SGet(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		if r.Method != http.MethodGet {
			t.Fatalf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	a := &TaboolaAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		CampaignID:  uuid.New(),
		TBLCI:       "tab-click-abc",
		EventType:   "registration",
		PayoutMicro: 2_500_000,
		TxID:        "order-9",
	}, srv.URL, "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(gotURL, "click-id=tab-click-abc") {
		t.Fatalf("url %q", gotURL)
	}
	if !strings.Contains(gotURL, "name=registration") {
		t.Fatalf("url %q", gotURL)
	}
	if !strings.Contains(gotURL, "revenue=2.50") {
		t.Fatalf("url %q", gotURL)
	}
	if !strings.Contains(gotURL, "orderid=order-9") {
		t.Fatalf("url %q", gotURL)
	}
}

func TestTaboolaAdapter_MissingClickID(t *testing.T) {
	a := &TaboolaAdapter{}
	err := a.Send(context.Background(), httptest.NewServer(nil).Client(), &PostbackPayload{}, "purchase", "")
	if err == nil || !strings.Contains(err.Error(), "click-id") {
		t.Fatalf("err=%v", err)
	}
}

func TestOutbrainAdapter_S2SGet(t *testing.T) {
	var gotURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		if r.Method != http.MethodGet {
			t.Fatalf("method %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &OutbrainAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		OBClickID:   "ob-v1-xyz",
		EventType:   "Purchase",
		PayoutMicro: 1_000_000,
	}, srv.URL, "")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(gotURL, "ob_click_id=ob-v1-xyz") {
		t.Fatalf("url %q", gotURL)
	}
	if !strings.Contains(gotURL, "name=Purchase") {
		t.Fatalf("url %q", gotURL)
	}
}

func TestMicrosoftAdsAdapter_OfflineConversion(t *testing.T) {
	var body microsoftAdsOfflinePayload
	var authToken, devToken, accountID, customerID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken = r.Header.Get("AuthenticationToken")
		devToken = r.Header.Get("DeveloperToken")
		accountID = r.Header.Get("CustomerAccountId")
		customerID = r.Header.Get("CustomerId")
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &MicrosoftAdsAdapter{}
	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	err := a.Send(context.Background(), client, &PostbackPayload{
		MSCLKID:       "msclkid-42",
		PayoutMicro:   3_000_000,
		TestEventCode: "dev-token-1",
	}, "111|222|My Offline Goal", "oauth-access")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if authToken != "oauth-access" || devToken != "dev-token-1" {
		t.Fatalf("headers auth=%q dev=%q", authToken, devToken)
	}
	if accountID != "111" || customerID != "222" {
		t.Fatalf("account=%q customer=%q", accountID, customerID)
	}
	if len(body.OfflineConversions) != 1 {
		t.Fatalf("conversions %+v", body.OfflineConversions)
	}
	conv := body.OfflineConversions[0]
	if conv.MicrosoftClickID != "msclkid-42" || conv.ConversionName != "My Offline Goal" {
		t.Fatalf("conv %+v", conv)
	}
	if conv.ConversionValue != 3.0 || conv.ConversionCurrencyCode != "USD" {
		t.Fatalf("value %+v", conv)
	}
}

func TestParseMicrosoftAdsConfig(t *testing.T) {
	cfg, err := parseMicrosoftAdsConfig("10|20|Lead")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.AccountID != "10" || cfg.CustomerID != "20" || cfg.ConversionName != "Lead" {
		t.Fatalf("cfg %+v", cfg)
	}
	if _, err := parseMicrosoftAdsConfig("bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestBuildPostbackPayloadFromEvent_NativeClickIDs(t *testing.T) {
	cust := uuid.New()
	evt := &domain.Event{
		CampaignID:         uuid.New(),
		ClickID:            "clk",
		Type:               "conversion",
		ClearingPriceMicro: 1_000_000,
		Payload:            []byte(`{"tblci":"t1","ob_click_id":"o1","msclkid":"m1"}`),
	}
	pb := buildPostbackPayloadFromEvent(evt, cust)
	if pb.TBLCI != "t1" || pb.OBClickID != "o1" || pb.MSCLKID != "m1" {
		t.Fatalf("payload %+v", pb)
	}
}

func roundTripRewriteHost(target string, base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		req = req.Clone(req.Context())
		req.URL.Scheme = "http"
		req.URL.Host = strings.TrimPrefix(target, "http://")
		return base.RoundTrip(req)
	})
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
