package postback

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/database"
)

const (
	googleAdsDefaultAPIVersion = "v16"
	googleAdsDefaultBaseURL    = "https://googleads.googleapis.com/v16"
)

type GoogleAdapter struct{}

type googleAdsConfig struct {
	CustomerID       string
	ConversionAction string
	APIVersion       string
	CustomEndpoint   string
}

type googleClickConversion struct {
	Gclid              string  `json:"gclid"`
	ConversionAction   string  `json:"conversionAction"`
	ConversionDateTime string  `json:"conversionDateTime"`
	ConversionValue    float64 `json:"conversionValue,omitempty"`
	CurrencyCode       string  `json:"currencyCode,omitempty"`
	OrderID            string  `json:"orderId,omitempty"`
}

type googleUploadClickConversionsRequest struct {
	Conversions    []googleClickConversion `json:"conversions"`
	PartialFailure bool                    `json:"partialFailure"`
}

func ValidateGooglePostbackConfig(urlTemplate string) error {
	cfg, err := parseGoogleAdsConfig(urlTemplate)
	if err != nil {
		return err
	}
	if cfg.CustomEndpoint != "" {
		return nil
	}
	if cfg.CustomerID == "" || cfg.CustomerID == "default" {
		return fmt.Errorf("google: customer_id is required")
	}
	if cfg.ConversionAction == "" {
		return fmt.Errorf("google: conversion_action_id is required")
	}
	return nil
}

func parseGoogleAdsConfig(urlTemplate string) (googleAdsConfig, error) {
	t := strings.TrimSpace(urlTemplate)
	if t == "" {
		return googleAdsConfig{}, fmt.Errorf("google: url_template required (customer_id|conversion_action_id or customers/.../conversionActions/...)")
	}
	if strings.HasPrefix(t, "http") {
		return googleAdsConfig{CustomEndpoint: t}, nil
	}
	if strings.HasPrefix(t, "customers/") {
		parts := strings.Split(t, "/")
		if len(parts) < 4 || parts[0] != "customers" || parts[2] != "conversionActions" {
			return googleAdsConfig{}, fmt.Errorf("google: invalid conversion action resource %q", t)
		}
		customerID := strings.TrimSpace(parts[1])
		if customerID == "" || customerID == "default" {
			return googleAdsConfig{}, fmt.Errorf("google: invalid customer_id in resource name")
		}
		if !database.ValidGAQLDigits(customerID) {
			return googleAdsConfig{}, fmt.Errorf("google: invalid customer_id %q", customerID)
		}
		return googleAdsConfig{
			CustomerID:       customerID,
			ConversionAction: t,
			APIVersion:       googleAdsDefaultAPIVersion,
		}, nil
	}

	parts := strings.Split(t, "|")
	if len(parts) < 2 || len(parts) > 3 {
		return googleAdsConfig{}, fmt.Errorf("google: url_template must be customer_id|conversion_action_id or full conversionActions resource path")
	}
	customerID := strings.TrimSpace(parts[0])
	actionPart := strings.TrimSpace(parts[1])
	if customerID == "" || customerID == "default" {
		return googleAdsConfig{}, fmt.Errorf("google: customer_id is required")
	}
	if !database.ValidGAQLDigits(customerID) {
		return googleAdsConfig{}, fmt.Errorf("google: invalid customer_id %q", customerID)
	}

	apiVersion := googleAdsDefaultAPIVersion
	if len(parts) == 3 {
		apiVersion = strings.TrimSpace(parts[2])
		if apiVersion == "" {
			return googleAdsConfig{}, fmt.Errorf("google: api version must not be empty")
		}
		if !strings.HasPrefix(apiVersion, "v") {
			apiVersion = "v" + apiVersion
		}
	}

	var actionResource string
	switch {
	case strings.HasPrefix(actionPart, "customers/"):
		actionResource = actionPart
	case database.ValidGAQLDigits(actionPart):
		actionResource = fmt.Sprintf("customers/%s/conversionActions/%s", customerID, actionPart)
	default:
		return googleAdsConfig{}, fmt.Errorf("google: conversion_action_id must be numeric or full resource name")
	}

	return googleAdsConfig{
		CustomerID:       customerID,
		ConversionAction: actionResource,
		APIVersion:       apiVersion,
	}, nil
}

func googleUploadEndpoint(cfg googleAdsConfig) string {
	if cfg.CustomEndpoint != "" {
		return cfg.CustomEndpoint
	}
	base := googleAdsDefaultBaseURL
	if cfg.APIVersion != "" && cfg.APIVersion != googleAdsDefaultAPIVersion {
		base = fmt.Sprintf("https://googleads.googleapis.com/%s", strings.TrimPrefix(cfg.APIVersion, "/"))
	}
	return fmt.Sprintf("%s/customers/%s:uploadClickConversions", strings.TrimRight(base, "/"), cfg.CustomerID)
}

func (a *GoogleAdapter) Send(ctx context.Context, client *http.Client, payload *PostbackPayload, urlTemplate, apiTokenDecrypted string) error {
	gclid := strings.TrimSpace(payload.GCLID)
	if gclid == "" {
		return fmt.Errorf("google: missing gclid on conversion payload")
	}

	cfg, err := parseGoogleAdsConfig(urlTemplate)
	if err != nil {
		return err
	}

	accessToken, developerToken, loginCustomerID, err := resolveGoogleAccessToken(ctx, client, apiTokenDecrypted, payload.TestEventCode)
	if err != nil {
		return err
	}
	if developerToken == "" && cfg.CustomEndpoint == "" {
		return fmt.Errorf("google: developer token required (test_event_code or api_token JSON)")
	}

	conv := googleClickConversion{
		Gclid:              gclid,
		ConversionAction:   cfg.ConversionAction,
		ConversionDateTime: time.Now().UTC().Format("2006-01-02 15:04:05+00:00"),
		OrderID:            ResolveEventID(payload),
	}
	if payload.PayoutMicro > 0 {
		conv.ConversionValue = payload.PayoutDollarsAPI()
		conv.CurrencyCode = "USD"
	}

	bodyBytes, err := json.Marshal(googleUploadClickConversionsRequest{
		Conversions:    []googleClickConversion{conv},
		PartialFailure: true,
	})
	if err != nil {
		return fmt.Errorf("google: marshal payload: %w", err)
	}

	endpoint := googleUploadEndpoint(cfg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("google: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	if developerToken != "" {
		req.Header.Set("developer-token", developerToken)
	}
	if loginCustomerID != "" {
		req.Header.Set("login-customer-id", loginCustomerID)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("google: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, readErr := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if readErr != nil {
		return fmt.Errorf("google: read response: %w", readErr)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var partial struct {
			PartialFailureError struct {
				Message string `json:"message"`
			} `json:"partialFailureError"`
		}
		if json.Unmarshal(body, &partial) == nil && strings.TrimSpace(partial.PartialFailureError.Message) != "" {
			return &DispatchHTTPError{
				StatusCode: http.StatusBadRequest,
				Body:       partial.PartialFailureError.Message,
			}
		}
		return nil
	}
	return &DispatchHTTPError{StatusCode: resp.StatusCode, Body: string(body)}
}
