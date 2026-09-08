package fraudadmin_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/pkg/httpresponse"

	"github.com/stretchr/testify/require"
)

type moderatorCorpusStub struct {
	listItems  []fraudadmin.ModeratorCorpusDTO
	upsertReq  fraudadmin.ModeratorCorpusUpsertRequest
	importCSV  string
	previewJA3 string
}

func (s *moderatorCorpusStub) ListTuples(_ context.Context, limit, offset int) ([]fraudadmin.ModeratorCorpusDTO, int64, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	end := offset + limit
	if end > len(s.listItems) {
		end = len(s.listItems)
	}
	start := offset
	if start > len(s.listItems) {
		start = len(s.listItems)
	}
	return s.listItems[start:end], int64(len(s.listItems)), nil
}

func (s *moderatorCorpusStub) UpsertTuple(_ context.Context, req fraudadmin.ModeratorCorpusUpsertRequest) (fraudadmin.ModeratorCorpusDTO, error) {
	s.upsertReq = req
	return fraudadmin.ModeratorCorpusDTO{
		ID:  "550e8400-e29b-41d4-a716-446655440099",
		JA3: req.JA3,
		JA4: req.JA4,
	}, nil
}

func (s *moderatorCorpusStub) ImportCSV(_ context.Context, csvBody string) (int, error) {
	s.importCSV = csvBody
	return 1, nil
}

func (s *moderatorCorpusStub) PreviewMatchCount7d(_ context.Context, ja3 string) (int64, error) {
	s.previewJA3 = ja3
	return 3, nil
}

func (s *moderatorCorpusStub) FeedLastRefresh(_ context.Context) (string, bool) {
	return "2026-09-08T10:00:00Z", true
}

func newModeratorCorpusHandlers(stub *moderatorCorpusStub, writeAllowed bool) *fraudadmin.HTTPHandlers {
	return &fraudadmin.HTTPHandlers{
		ModeratorCorpus: stub,
		RequirePermission: func(permission string, next http.HandlerFunc) http.HandlerFunc {
			if permission != "audit:read" {
				return func(w http.ResponseWriter, _ *http.Request) {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
				}
			}
			return next
		},
		RequireAnyPermission: func(_ []string, next http.HandlerFunc) http.HandlerFunc {
			if !writeAllowed {
				return func(w http.ResponseWriter, _ *http.Request) {
					httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
				}
			}
			return next
		},
	}
}

func TestListModeratorCorpus_ok(t *testing.T) {
	stub := &moderatorCorpusStub{
		listItems: []fraudadmin.ModeratorCorpusDTO{{ID: "1", JA3: "771,4865"}},
	}
	h := newModeratorCorpusHandlers(stub, true)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/fraud/moderator-corpus", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var body fraudadmin.ModeratorCorpusListResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Items, 1)
	require.Equal(t, "771,4865", body.Items[0].JA3)
}

func TestPostModeratorCorpus_writeDenied_holdout(t *testing.T) {
	stub := &moderatorCorpusStub{}
	h := newModeratorCorpusHandlers(stub, false)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fraud/moderator-corpus", strings.NewReader(`{"ja3":"771,4865"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Empty(t, stub.upsertReq.JA3)
}

func TestPostModeratorCorpus_upsert(t *testing.T) {
	stub := &moderatorCorpusStub{}
	h := newModeratorCorpusHandlers(stub, true)
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/fraud/moderator-corpus", strings.NewReader(`{"ja3":"771,4865-4866"}`))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "771,4865-4866", stub.upsertReq.JA3)
}
