package fraudadmin

import (
	"net/http"
	"strconv"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) registerModeratorCorpusRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc, permAny func([]string, http.HandlerFunc) http.HandlerFunc) {
	if h == nil || h.ModeratorCorpus == nil {
		return
	}
	mux.HandleFunc("GET /api/v1/fraud/moderator-corpus", limit(perm("audit:read", h.listModeratorCorpus)))
	mux.HandleFunc("POST /api/v1/fraud/moderator-corpus", limit(permAny([]string{"campaigns:write", "shards:write"}, h.postModeratorCorpus)))
	mux.HandleFunc("POST /api/v1/fraud/moderator-corpus/import", limit(permAny([]string{"campaigns:write", "shards:write"}, h.postModeratorCorpusImport)))
	mux.HandleFunc("GET /api/v1/fraud/moderator-corpus/preview", limit(perm("audit:read", h.getModeratorCorpusPreview)))
}

func (h *HTTPHandlers) listModeratorCorpus(w http.ResponseWriter, r *http.Request) {
	limit := ModeratorCorpusDefaultLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid limit")
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("offset")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid offset")
			return
		}
		offset = parsed
	}
	items, total, err := h.ModeratorCorpus.ListTuples(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if items == nil {
		items = []ModeratorCorpusDTO{}
	}
	resp := ModeratorCorpusListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
	if ts, ok := h.ModeratorCorpus.FeedLastRefresh(r.Context()); ok {
		resp.LastRefresh = ts
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *HTTPHandlers) postModeratorCorpus(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[ModeratorCorpusUpsertRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	dto, err := h.ModeratorCorpus.UpsertTuple(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *HTTPHandlers) postModeratorCorpusImport(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[ModeratorCorpusImportRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if strings.TrimSpace(req.CSV) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "csv is required")
		return
	}
	upserted, err := h.ModeratorCorpus.ImportCSV(r.Context(), req.CSV)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, ModeratorCorpusImportResponse{Upserted: upserted})
}

func (h *HTTPHandlers) getModeratorCorpusPreview(w http.ResponseWriter, r *http.Request) {
	ja3 := strings.TrimSpace(r.URL.Query().Get("ja3"))
	if ja3 == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ja3 query param is required")
		return
	}
	count, err := h.ModeratorCorpus.PreviewMatchCount7d(r.Context(), ja3)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, ModeratorCorpusPreviewResponse{MatchCount7d: count})
}
