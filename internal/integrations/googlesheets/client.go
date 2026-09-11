package googlesheets

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

type Client struct {
	cfg    Config
	store  *Store
	client *http.Client
}

func NewClient(cfg Config, store *Store) *Client {
	return &Client{
		cfg:    cfg,
		store:  store,
		client: &http.Client{Timeout: defaultSheetHTTPTimeout},
	}
}

func (c *Client) ResolveAccessToken(ctx context.Context, userID uuid.UUID) (string, error) {
	if c == nil || c.store == nil {
		return "", fmt.Errorf("google sheets client unavailable")
	}
	row, err := c.store.LoadTokenRow(ctx, userID)
	if err != nil {
		return "", err
	}
	access := strings.TrimSpace(row.AccessToken)
	needsRefresh := access == "" ||
		(!row.ExpiresAt.IsZero() && time.Now().After(row.ExpiresAt.Add(-30*time.Second)))
	if needsRefresh {
		refreshed, _, expires, refreshErr := refreshAccessToken(ctx, c.client, c.cfg, row.RefreshToken)
		if refreshErr != nil {
			if access == "" {
				return "", refreshErr
			}
		} else if refreshed != "" {
			access = refreshed
			if err := c.store.UpsertTokens(ctx, userID, row.RefreshToken, access, expires, row.Scopes); err != nil {
				return "", err
			}
		}
	}
	if access == "" {
		return "", ErrNotConnected
	}
	return access, nil
}

type ExportOptions struct {
	Mode          string
	SpreadsheetID string
	SheetTitle    string
	ReportTitle   string
	CSVPath       string
}

func (c *Client) UploadCSVFile(ctx context.Context, userID uuid.UUID, opts ExportOptions) (UploadResult, error) {
	access, err := c.ResolveAccessToken(ctx, userID)
	if err != nil {
		return UploadResult{}, err
	}
	csvFile, err := os.Open(opts.CSVPath)
	if err != nil {
		return UploadResult{}, err
	}
	defer func() { _ = csvFile.Close() }()
	mode := strings.TrimSpace(opts.Mode)
	if mode == "" {
		mode = "create"
	}
	tabTitle := strings.TrimSpace(opts.SheetTitle)
	if tabTitle == "" {
		tabTitle = defaultTabTitle(opts.ReportTitle)
	}
	switch mode {
	case "append":
		spreadsheetID := strings.TrimSpace(opts.SpreadsheetID)
		if spreadsheetID == "" {
			return UploadResult{}, fmt.Errorf("spreadsheet_id required for append mode")
		}
		if err := c.addSheetTab(ctx, access, spreadsheetID, tabTitle); err != nil {
			return UploadResult{}, err
		}
		writer := newBatchWriter(c.client, access, spreadsheetID, tabTitle)
		if err := writer.WriteRowsFromCSV(ctx, csvFile); err != nil {
			return UploadResult{}, err
		}
		return UploadResult{
			SpreadsheetID:  spreadsheetID,
			SpreadsheetURL: spreadsheetURL(spreadsheetID),
		}, nil
	default:
		title := strings.TrimSpace(opts.ReportTitle)
		if title == "" {
			title = "Report export"
		}
		spreadsheetID, err := c.createSpreadsheet(ctx, access, title, tabTitle)
		if err != nil {
			return UploadResult{}, err
		}
		writer := newBatchWriter(c.client, access, spreadsheetID, tabTitle)
		if err := writer.WriteRowsFromCSV(ctx, csvFile); err != nil {
			return UploadResult{}, err
		}
		return UploadResult{
			SpreadsheetID:  spreadsheetID,
			SpreadsheetURL: spreadsheetURL(spreadsheetID),
		}, nil
	}
}

func (c *Client) createSpreadsheet(ctx context.Context, accessToken, title, tabTitle string) (string, error) {
	body := map[string]any{
		"properties": map[string]any{"title": title},
		"sheets": []map[string]any{
			{"properties": map[string]any{"title": tabTitle}},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://sheets.googleapis.com/v4/spreadsheets", strings.NewReader(string(raw)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return "", err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 8192))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("google sheets create: status %d: %s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		SpreadsheetID string `json:"spreadsheetId"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return "", err
	}
	if parsed.SpreadsheetID == "" {
		return "", fmt.Errorf("google sheets create: missing spreadsheet id")
	}
	return parsed.SpreadsheetID, nil
}

func (c *Client) addSheetTab(ctx context.Context, accessToken, spreadsheetID, tabTitle string) error {
	body := map[string]any{
		"requests": []map[string]any{
			{
				"addSheet": map[string]any{
					"properties": map[string]any{"title": tabTitle},
				},
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://sheets.googleapis.com/v4/spreadsheets/%s:batchUpdate", spreadsheetID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		coldpath.CloseHTTPResponse(resp)
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("google sheets add tab: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func defaultTabTitle(reportTitle string) string {
	base := strings.TrimSpace(reportTitle)
	if base == "" {
		base = "Export"
	}
	if len(base) > 80 {
		base = base[:80]
	}
	return base + " " + time.Now().UTC().Format("2006-01-02")
}

func spreadsheetURL(spreadsheetID string) string {
	return "https://docs.google.com/spreadsheets/d/" + spreadsheetID
}

func RedirectURI(publicURL string) string {
	base := strings.TrimRight(strings.TrimSpace(publicURL), "/")
	return base + "/api/v1/integrations/google-sheets/callback"
}

func ConnectURL(cfg Config, state string) (string, error) {
	if cfg.ClientID == "" {
		return "", ErrNotConfigured
	}
	redirectURI := RedirectURI(cfg.PublicURL)
	values := url.Values{}
	values.Set("client_id", cfg.ClientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("response_type", "code")
	values.Set("scope", OAuthScope)
	values.Set("access_type", "offline")
	values.Set("prompt", "consent")
	values.Set("state", state)
	return "https://accounts.google.com/o/oauth2/v2/auth?" + values.Encode(), nil
}
