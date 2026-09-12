package googlesheets

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
)

type batchWriter struct {
	client    *http.Client
	accessTok string
	sheetID   string
	tabTitle  string
	apiBase   string
}

func newBatchWriter(client *http.Client, accessToken, spreadsheetID, tabTitle string) *batchWriter {
	if client == nil {
		client = &http.Client{Timeout: defaultSheetHTTPTimeout}
	}
	return &batchWriter{
		client:    client,
		accessTok: accessToken,
		sheetID:   spreadsheetID,
		tabTitle:  tabTitle,
	}
}

func (w *batchWriter) WriteRows(ctx context.Context, rows [][]string) error {
	if w == nil || w.client == nil {
		return fmt.Errorf("sheets batch writer unavailable")
	}
	if len(rows) == 0 {
		return nil
	}
	colCount := maxRowWidth(rows)
	if colCount == 0 {
		return nil
	}
	maxRowsPerBatch := rowsPerBatch(colCount)
	return w.writeRowBatches(ctx, rows, maxRowsPerBatch)
}

func (w *batchWriter) WriteRowsFromCSV(ctx context.Context, r io.Reader) error {
	if w == nil || w.client == nil {
		return fmt.Errorf("sheets batch writer unavailable")
	}
	csvReader := csv.NewReader(r)
	first, err := csvReader.Read()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	colCount := len(first)
	if colCount == 0 {
		return nil
	}
	maxRowsPerBatch := rowsPerBatch(colCount)
	startRow := 1
	batch := make([][]string, 0, maxRowsPerBatch)
	batch = append(batch, first)
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		rangeA1 := fmt.Sprintf("'%s'!A%d", escapeSheetTitle(w.tabTitle), startRow)
		if err := w.putValues(ctx, rangeA1, batch); err != nil {
			return err
		}
		startRow += len(batch)
		batch = batch[:0]
		return nil
	}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		batch = append(batch, record)
		if len(batch) >= maxRowsPerBatch {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

func maxRowWidth(rows [][]string) int {
	colCount := 0
	for _, row := range rows {
		if len(row) > colCount {
			colCount = len(row)
		}
	}
	return colCount
}

func rowsPerBatch(colCount int) int {
	if colCount <= 0 {
		return 1
	}
	maxRowsPerBatch := maxCellsPerBatch / colCount
	if maxRowsPerBatch < 1 {
		maxRowsPerBatch = 1
	}
	return maxRowsPerBatch
}

func (w *batchWriter) writeRowBatches(ctx context.Context, rows [][]string, maxRowsPerBatch int) error {
	startRow := 1
	for offset := 0; offset < len(rows); offset += maxRowsPerBatch {
		end := offset + maxRowsPerBatch
		if end > len(rows) {
			end = len(rows)
		}
		chunk := rows[offset:end]
		rangeA1 := fmt.Sprintf("'%s'!A%d", escapeSheetTitle(w.tabTitle), startRow)
		if err := w.putValues(ctx, rangeA1, chunk); err != nil {
			return err
		}
		startRow += len(chunk)
	}
	return nil
}

func (w *batchWriter) putValues(ctx context.Context, rangeA1 string, values [][]string) error {
	body := map[string]any{
		"valueInputOption": "RAW",
		"data": []map[string]any{
			{
				"range":  rangeA1,
				"values": values,
			},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return w.doWithBackoff(ctx, http.MethodPost, w.valuesBatchUpdateURL(), raw)
}

func (w *batchWriter) valuesBatchUpdateURL() string {
	base := strings.TrimRight(strings.TrimSpace(w.apiBase), "/")
	if base == "" {
		base = "https://sheets.googleapis.com/v4"
	}
	return fmt.Sprintf("%s/spreadsheets/%s/values:batchUpdate", base, w.sheetID)
}

func (w *batchWriter) doWithBackoff(ctx context.Context, method, url string, body []byte) error {
	backoff := 500 * time.Millisecond
	for attempt := range 5 {
		req, err := http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(body)))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+w.accessTok)
		req.Header.Set("Content-Type", "application/json")
		resp, err := w.client.Do(req)
		if err != nil {
			coldpath.CloseHTTPResponse(resp)
			return err
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
		_ = resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			if attempt == 4 {
				return fmt.Errorf("google sheets api: status %d: %s", resp.StatusCode, string(respBody))
			}
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			backoff *= 2
			continue
		}
		return fmt.Errorf("google sheets api: status %d: %s", resp.StatusCode, string(respBody))
	}
	return fmt.Errorf("google sheets api: retries exhausted")
}

func escapeSheetTitle(title string) string {
	return strings.ReplaceAll(title, "'", "''")
}
