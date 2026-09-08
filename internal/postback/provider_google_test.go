package postback

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseGoogleAdsConfig(t *testing.T) {
	cfg, err := parseGoogleAdsConfig("1234567890|987654321")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.CustomerID != "1234567890" {
		t.Fatalf("customer_id=%q", cfg.CustomerID)
	}
	if cfg.ConversionAction != "customers/1234567890/conversionActions/987654321" {
		t.Fatalf("action=%q", cfg.ConversionAction)
	}

	cfg, err = parseGoogleAdsConfig("customers/111/conversionActions/222")
	if err != nil {
		t.Fatalf("resource parse: %v", err)
	}
	if cfg.CustomerID != "111" || cfg.ConversionAction != "customers/111/conversionActions/222" {
		t.Fatalf("cfg %+v", cfg)
	}

	cfg, err = parseGoogleAdsConfig("1234567890|customers/1234567890/conversionActions/42|17")
	if err != nil {
		t.Fatalf("version parse: %v", err)
	}
	if cfg.APIVersion != "v17" {
		t.Fatalf("api_version=%q", cfg.APIVersion)
	}

	if _, err := parseGoogleAdsConfig("default|123"); err == nil {
		t.Fatal("expected error for default customer")
	}
	if _, err := parseGoogleAdsConfig("bad"); err == nil {
		t.Fatal("expected error for malformed template")
	}
}

func TestValidateGooglePostbackConfig(t *testing.T) {
	if err := ValidateGooglePostbackConfig("1234567890|987654321"); err != nil {
		t.Fatalf("valid config: %v", err)
	}
	if err := ValidateGooglePostbackConfig("https://example.com/upload"); err != nil {
		t.Fatalf("custom endpoint: %v", err)
	}
	if err := ValidateGooglePostbackConfig("default|1"); err == nil {
		t.Fatal("expected rejection of default customer")
	}
}

func TestGoogleAdapter_UploadClickConversions(t *testing.T) {
	var body googleUploadClickConversionsRequest
	var authToken, devToken, loginCustomerID string
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authToken = r.Header.Get("Authorization")
		devToken = r.Header.Get("developer-token")
		loginCustomerID = r.Header.Get("login-customer-id")
		gotPath = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("unmarshal: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"results":[{}]}`))
	}))
	defer srv.Close()

	a := &GoogleAdapter{}
	client := &http.Client{Transport: roundTripRewriteHost(srv.URL, nil)}
	err := a.Send(context.Background(), client, &PostbackPayload{
		GCLID:         "gclid-42",
		PayoutMicro:   5_000_000,
		EventID:       "evt-google-1",
		TestEventCode: "dev-token-1",
	}, "1234567890|987654321", `{"access_token":"oauth-access","login_customer_id":"999"}`)
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if authToken != "Bearer oauth-access" {
		t.Fatalf("auth=%q", authToken)
	}
	if devToken != "dev-token-1" || loginCustomerID != "999" {
		t.Fatalf("dev=%q login=%q", devToken, loginCustomerID)
	}
	if !strings.Contains(gotPath, "/customers/1234567890:uploadClickConversions") {
		t.Fatalf("path %q", gotPath)
	}
	if len(body.Conversions) != 1 {
		t.Fatalf("conversions %+v", body.Conversions)
	}
	conv := body.Conversions[0]
	if conv.Gclid != "gclid-42" || conv.OrderID != "evt-google-1" {
		t.Fatalf("conv %+v", conv)
	}
	if conv.ConversionAction != "customers/1234567890/conversionActions/987654321" {
		t.Fatalf("action=%q", conv.ConversionAction)
	}
	if conv.ConversionValue != 5.0 || conv.CurrencyCode != "USD" {
		t.Fatalf("value %+v", conv)
	}
	if !body.PartialFailure {
		t.Fatal("expected partialFailure=true")
	}
}

func TestGoogleAdapter_CustomEndpoint(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	a := &GoogleAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		GCLID: "gclid-1",
	}, srv.URL, "oauth-only")
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method %q", gotMethod)
	}
}

func TestGoogleAdapter_MissingGclid(t *testing.T) {
	a := &GoogleAdapter{}
	err := a.Send(context.Background(), httptest.NewServer(nil).Client(), &PostbackPayload{}, "123|456", "tok")
	if err == nil || !strings.Contains(err.Error(), "gclid") {
		t.Fatalf("err=%v", err)
	}
}

func TestGoogleAdapter_APIErrorSurfacesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"message":"Customer not found","status":"PERMISSION_DENIED"}}`))
	}))
	defer srv.Close()

	a := &GoogleAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		GCLID:         "gclid-err",
		TestEventCode: "dev",
	}, srv.URL, "oauth")
	var httpErr *DispatchHTTPError
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected DispatchHTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 403 || !strings.Contains(httpErr.Body, "PERMISSION_DENIED") {
		t.Fatalf("err=%v", err)
	}
	if !httpErr.Permanent() {
		t.Fatalf("expected permanent 4xx error, got %v", err)
	}
}

func TestGoogleAdapter_PartialFailureOn200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"partialFailureError":{"code":3,"message":"The click ID is invalid."}}`))
	}))
	defer srv.Close()

	a := &GoogleAdapter{}
	err := a.Send(context.Background(), srv.Client(), &PostbackPayload{
		GCLID:         "gclid-bad",
		TestEventCode: "dev",
	}, srv.URL, `{"access_token":"oauth"}`)
	var httpErr *DispatchHTTPError
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected DispatchHTTPError, got %T: %v", err, err)
	}
	if httpErr.StatusCode != 400 || !strings.Contains(httpErr.Body, "click ID") {
		t.Fatalf("err=%v", err)
	}
}

func TestRefreshGoogleOAuth(t *testing.T) {
	var gotRefresh string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotRefresh = string(body)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "fresh-token",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	client := srv.Client()
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.Transport = roundTripRewriteHost(srv.URL, transport)

	token, _, err := refreshGoogleOAuth(context.Background(), client, "client-id", "client-secret", "refresh-abc")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if token != "fresh-token" || !strings.Contains(gotRefresh, "refresh-abc") {
		t.Fatalf("token=%q refresh=%q", token, gotRefresh)
	}
}
