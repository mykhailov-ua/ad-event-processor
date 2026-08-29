package provider

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/money"
)

const (
	microsoftAdsReportBaseDefault = "https://reporting.api.bingads.microsoft.com/Reporting/v13"
	microsoftAdsPollMaxAttempts   = 30
	microsoftAdsPollInterval      = 2 * time.Second
)

type microsoftAdsSubmitResponse struct {
	ReportRequestID string `json:"ReportRequestId"`
}

type microsoftAdsPollResponse struct {
	ReportRequestStatus struct {
		ReportDownloadURL string `json:"ReportDownloadUrl"`
		Status            string `json:"Status"`
	} `json:"ReportRequestStatus"`
}

func fetchMicrosoftAdsCosts(ctx context.Context, client *http.Client, baseURL string, cred Credential, date time.Time) ([]CostLine, error) {
	base := baseURL
	if base == "" {
		base = microsoftAdsReportBaseDefault
	}
	if client == nil {
		client = &http.Client{Timeout: 120 * time.Second}
	}

	accountID, customerID, developerToken, err := microsoftAdsCredentialFields(cred)
	if err != nil {
		return nil, err
	}
	token := strings.TrimSpace(cred.AccessToken)
	if token == "" {
		return nil, fmt.Errorf("microsoft_ads: missing access token")
	}

	dateStr := date.UTC().Format("2006-01-02")
	if !database.ValidGAQLDate(dateStr) {
		return nil, fmt.Errorf("microsoft_ads: invalid date %q", dateStr)
	}

	accountNum, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("microsoft_ads: account id must be numeric: %w", err)
	}

	submitBody := map[string]any{
		"ReportRequest": map[string]any{
			"ExcludeColumnHeaders":   false,
			"ExcludeReportFooter":    true,
			"ExcludeReportHeader":    true,
			"Format":                 "Csv",
			"ReportName":             "Campaign Performance Report",
			"ReturnOnlyCompleteData": false,
			"Type":                   "CampaignPerformanceReportRequest",
			"Aggregation":            "Daily",
			"Columns":                []string{"CampaignId", "Spend"},
			"Scope": map[string]any{
				"AccountIds": []int64{accountNum},
			},
			"Time": map[string]any{
				"CustomDateRangeStart": map[string]int{
					"Day": date.Day(), "Month": int(date.Month()), "Year": date.Year(),
				},
				"CustomDateRangeEnd": map[string]int{
					"Day": date.Day(), "Month": int(date.Month()), "Year": date.Year(),
				},
				"ReportTimeZone": "GreenwichMeanTimeDublinEdinburghLisbonLondon",
			},
		},
	}
	payload, err := json.Marshal(submitBody)
	if err != nil {
		return nil, err
	}

	submitURL := strings.TrimRight(base, "/") + "/GenerateReport/Submit"
	submitReq, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	microsoftAdsSetHeaders(submitReq, token, customerID, accountID, developerToken)
	submitReq.Header.Set("Content-Type", "application/json")

	submitBodyBytes, submitStatus, err := doReadLimitedHTTPBody(client, submitReq, 1<<20)
	if err != nil {
		return nil, fmt.Errorf("microsoft_ads submit: %w", err)
	}
	if submitStatus != http.StatusOK {
		return nil, fmt.Errorf("microsoft_ads submit: status %d: %s", submitStatus, string(submitBodyBytes))
	}

	var submitted microsoftAdsSubmitResponse
	if err := json.Unmarshal(submitBodyBytes, &submitted); err != nil {
		return nil, err
	}
	if submitted.ReportRequestID == "" {
		return nil, fmt.Errorf("microsoft_ads submit: empty ReportRequestId")
	}

	downloadURL, err := microsoftAdsPollReport(ctx, client, base, submitted.ReportRequestID, token, customerID, accountID, developerToken)
	if err != nil {
		return nil, err
	}

	reportBytes, err := microsoftAdsDownloadReport(ctx, client, downloadURL)
	if err != nil {
		return nil, err
	}
	csvBytes, err := microsoftAdsDecompressReport(reportBytes)
	if err != nil {
		return nil, err
	}

	return parseMicrosoftAdsCampaignCSV(cred, date, csvBytes)
}

func microsoftAdsCredentialFields(cred Credential) (accountID, customerID, developerToken string, err error) {
	accountID = strings.TrimSpace(cred.AccountID)
	if accountID == "" {
		accountID = strings.TrimSpace(cred.ExtraConfig["customer_account_id"])
	}
	customerID = strings.TrimSpace(cred.ExtraConfig["customer_id"])
	developerToken = strings.TrimSpace(cred.ExtraConfig["developer_token"])
	if accountID == "" {
		return "", "", "", fmt.Errorf("microsoft_ads: missing account id")
	}
	if customerID == "" {
		return "", "", "", fmt.Errorf("microsoft_ads: missing customer_id in extra_config")
	}
	if developerToken == "" {
		return "", "", "", fmt.Errorf("microsoft_ads: missing developer_token in extra_config")
	}
	return accountID, customerID, developerToken, nil
}

func microsoftAdsSetHeaders(req *http.Request, token, customerID, accountID, developerToken string) {
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("CustomerId", customerID)
	req.Header.Set("CustomerAccountId", accountID)
	req.Header.Set("DeveloperToken", developerToken)
}

func microsoftAdsPollReport(ctx context.Context, client *http.Client, base, reportRequestID, token, customerID, accountID, developerToken string) (string, error) {
	pollURL := strings.TrimRight(base, "/") + "/GenerateReport/Poll"
	pollPayload, err := json.Marshal(map[string]string{"ReportRequestId": reportRequestID})
	if err != nil {
		return "", err
	}

	for attempt := range microsoftAdsPollMaxAttempts {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if attempt > 0 {
			timer := time.NewTimer(microsoftAdsPollInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return "", ctx.Err()
			case <-timer.C:
			}
		}

		pollReq, err := http.NewRequestWithContext(ctx, http.MethodPost, pollURL, bytes.NewReader(pollPayload))
		if err != nil {
			return "", err
		}
		microsoftAdsSetHeaders(pollReq, token, customerID, accountID, developerToken)
		pollReq.Header.Set("Content-Type", "application/json")

		body, pollStatus, err := doReadLimitedHTTPBody(client, pollReq, 1<<20)
		if err != nil {
			return "", fmt.Errorf("microsoft_ads poll: %w", err)
		}
		if pollStatus != http.StatusOK {
			return "", fmt.Errorf("microsoft_ads poll: status %d: %s", pollStatus, string(body))
		}

		var polled microsoftAdsPollResponse
		if err := json.Unmarshal(body, &polled); err != nil {
			return "", err
		}
		switch strings.ToLower(polled.ReportRequestStatus.Status) {
		case "success":
			if polled.ReportRequestStatus.ReportDownloadURL == "" {
				return "", fmt.Errorf("microsoft_ads poll: success without download url")
			}
			return polled.ReportRequestStatus.ReportDownloadURL, nil
		case "pending", "inprogress", "in_progress":
			continue
		case "error":
			return "", fmt.Errorf("microsoft_ads poll: report generation failed")
		default:
			if polled.ReportRequestStatus.ReportDownloadURL != "" {
				return polled.ReportRequestStatus.ReportDownloadURL, nil
			}
		}
	}
	return "", fmt.Errorf("microsoft_ads poll: timed out after %d attempts", microsoftAdsPollMaxAttempts)
}

func microsoftAdsDownloadReport(ctx context.Context, client *http.Client, downloadURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("microsoft_ads download: status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(io.LimitReader(resp.Body, 16<<20))
}

func microsoftAdsDecompressReport(data []byte) ([]byte, error) {
	if len(data) < 2 || data[0] != 'P' || data[1] != 'K' {
		return data, nil
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("microsoft_ads report zip: %w", err)
	}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		rc, err := file.Open()
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(rc, 16<<20))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		return raw, nil
	}
	return nil, fmt.Errorf("microsoft_ads report zip: empty archive")
}

func parseMicrosoftAdsCampaignCSV(cred Credential, date time.Time, raw []byte) ([]CostLine, error) {
	reader := csv.NewReader(bytes.NewReader(raw))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("microsoft_ads csv: %w", err)
	}

	campaignCol, spendCol := -1, -1
	dataStart := 0
	for i, row := range records {
		for j, col := range row {
			name := strings.Trim(strings.TrimSpace(col), `"`)
			switch strings.ToLower(name) {
			case "campaignid":
				campaignCol = j
			case "spend":
				spendCol = j
			}
		}
		if campaignCol >= 0 && spendCol >= 0 {
			dataStart = i + 1
			break
		}
	}
	if campaignCol < 0 || spendCol < 0 {
		return nil, fmt.Errorf("microsoft_ads csv: missing CampaignId or Spend column")
	}

	lines := make([]CostLine, 0)
	for _, row := range records[dataStart:] {
		if campaignCol >= len(row) || spendCol >= len(row) {
			continue
		}
		campaignID := strings.Trim(strings.TrimSpace(row[campaignCol]), `"`)
		if campaignID == "" {
			continue
		}
		spendMicro, err := money.ParseDecimal(strings.Trim(strings.TrimSpace(row[spendCol]), `"`))
		if err != nil || spendMicro == 0 {
			continue
		}
		lines = append(lines, CostLine{
			CustomerID:  cred.CustomerID,
			CampaignID:  uuid.NewSHA1(cred.CustomerID, []byte("microsoft_ads:"+campaignID)),
			Date:        date,
			Network:     "microsoft_ads",
			PlacementID: campaignID,
			LineType:    LineTypeSpend,
			AmountMicro: spendMicro,
			Currency:    "USD",
		})
	}
	return lines, nil
}
