package googlesheets

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchWriter_doWithBackoff_retries429(t *testing.T) {
	t.Parallel()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	writer := &batchWriter{
		client:    srv.Client(),
		accessTok: "token",
	}
	err := writer.doWithBackoff(context.Background(), http.MethodPost, srv.URL, []byte(`{"valueInputOption":"RAW"}`))
	require.NoError(t, err)
	require.Equal(t, 2, calls)
}

func TestEscapeSheetTitle_quotes(t *testing.T) {
	t.Parallel()
	require.Equal(t, "Tab''Name", escapeSheetTitle("Tab'Name"))
}

func TestBatchWriter_cellBudgetSplitsRows_holdout(t *testing.T) {
	t.Parallel()
	colCount := 3
	maxRowsPerBatch := maxCellsPerBatch / colCount
	require.Equal(t, 3333, maxRowsPerBatch)
	totalRows := 5001
	chunks := 0
	for offset := 0; offset < totalRows; offset += maxRowsPerBatch {
		chunks++
	}
	require.Equal(t, 2, chunks)
}

func TestBatchWriter_WriteRowsFromCSV_batchesHoldout(t *testing.T) {
	t.Parallel()

	var requests int
	var firstRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Contains(t, r.URL.Path, "/values:batchUpdate")
		requests++
		if requests == 1 {
			var payload struct {
				Data []struct {
					Range string `json:"range"`
				} `json:"data"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
			require.Len(t, payload.Data, 1)
			firstRange = payload.Data[0].Range
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	const dataRows = 5000
	var csvBuilder strings.Builder
	csvBuilder.WriteString("h1,h2,h3\n")
	for i := 0; i < dataRows; i++ {
		fmt.Fprintf(&csvBuilder, "v%d,v%d,v%d\n", i, i, i)
	}

	writer := newBatchWriter(srv.Client(), "token", "sheet-id", "Export Tab")
	writer.apiBase = srv.URL
	err := writer.WriteRowsFromCSV(context.Background(), strings.NewReader(csvBuilder.String()))
	require.NoError(t, err)
	require.Equal(t, 2, requests, "5001 rows at 3 cols must split into two batchUpdate calls")
	require.Equal(t, "'Export Tab'!A1", firstRange)
}

func TestBatchWriter_WriteRowsFromCSV_singleBatch_holdout(t *testing.T) {
	t.Parallel()

	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	csvData := "a,b,c\n1,2,3\n4,5,6\n"
	writer := newBatchWriter(srv.Client(), "token", "sheet-id", "Tab")
	writer.apiBase = srv.URL
	err := writer.WriteRowsFromCSV(context.Background(), strings.NewReader(csvData))
	require.NoError(t, err)
	require.Equal(t, 1, requests)
}

func TestBatchWriter_WriteRows_batchesHoldout(t *testing.T) {
	t.Parallel()

	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	colCount := 3
	maxRowsPerBatch := maxCellsPerBatch / colCount
	totalRows := maxRowsPerBatch + 1
	rows := make([][]string, 0, totalRows)
	rows = append(rows, []string{"h1", "h2", "h3"})
	for i := 0; i < maxRowsPerBatch; i++ {
		rows = append(rows, []string{"1", "2", "3"})
	}

	writer := newBatchWriter(srv.Client(), "token", "sheet-id", "Tab")
	writer.apiBase = srv.URL
	err := writer.WriteRows(context.Background(), rows)
	require.NoError(t, err)
	require.Equal(t, 2, requests)
}
