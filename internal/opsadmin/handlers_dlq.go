package opsadmin

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) listDLQ(w http.ResponseWriter, r *http.Request) {
	limit, _ := coldpath.ParseAPIPagination(r)
	result, err := h.OpsReader.ListDLQEntries(r.Context(), r.URL.Query().Get("cursor"), int(limit))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(result.Errors) > 0 && len(result.Items) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, result)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

type dlqRetryRequest struct {
	ShardID int    `json:"shard_id"`
	Stream  string `json:"stream"`
	EntryID string `json:"entry_id"`
}

func (h *HTTPHandlers) retryDLQ(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	dlqID := r.PathValue("id")
	var req dlqRetryRequest
	if stringsHasPrefixJSON(r) {
		body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
		if err != nil {
			return
		}
		if len(body) > 0 {
			decoded, decodeErr := coldpath.DecodeBody[dlqRetryRequest](body)
			if decodeErr != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
				return
			}
			req = decoded
		}
	}

	if req.EntryID == "" {
		req.EntryID = parseDLQEntryIDFromRoute(dlqID)
	}
	if req.ShardID == 0 {
		req.ShardID = parseDLQShardFromRoute(dlqID)
	}

	payload := DLQRetryPayload{
		ShardID: req.ShardID,
		Stream:  req.Stream,
		EntryID: req.EntryID,
		DLQID:   dlqID,
	}
	dedup := sha256.Sum256([]byte(dlqID + idempotencyKey))
	if err := h.OpsReader.EnqueueDLQRetry(r.Context(), payload, hex.EncodeToString(dedup[:])); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *HTTPHandlers) listDLQInbox(w http.ResponseWriter, r *http.Request) {
	limit, _ := coldpath.ParseAPIPagination(r)
	result, err := h.OpsReader.ListDLQInbox(r.Context(), r.URL.Query().Get("source"), r.URL.Query().Get("cursor"), int(limit))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if len(result.Errors) > 0 && len(result.Items) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, result)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

type dlqInboxRetryRequest struct {
	Source string `json:"source"`
}

func (h *HTTPHandlers) retryDLQInbox(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if stringsHasPrefixJSON(r) {
		body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
		if err != nil {
			return
		}
		if len(body) > 0 {
			decoded, decodeErr := coldpath.DecodeBody[dlqInboxRetryRequest](body)
			if decodeErr != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
				return
			}
			if source == "" {
				source = decoded.Source
			}
		}
	}
	if source == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source is required")
		return
	}
	dlqID := r.PathValue("id")
	dedup := sha256.Sum256([]byte(source + ":" + dlqID + ":" + idempotencyKey))
	if err := h.OpsReader.RetryDLQInbox(r.Context(), source, dlqID, hex.EncodeToString(dedup[:])); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *HTTPHandlers) ListDLQInbox(w http.ResponseWriter, r *http.Request) {
	h.listDLQInbox(w, r)
}

func (h *HTTPHandlers) RetryDLQInbox(w http.ResponseWriter, r *http.Request) {
	h.retryDLQInbox(w, r)
}
