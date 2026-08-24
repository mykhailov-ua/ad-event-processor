package controlplane

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type PublisherService interface {
	ResolvePublisherBind(ctx context.Context, userID uuid.UUID) (PublisherBind, error)
	GetPublisherDashboard(ctx context.Context, bind PublisherBind, from, to time.Time) (PublisherDashboardDTO, error)
	ListPublisherStatements(ctx context.Context, bind PublisherBind, from, to time.Time, limit, offset int32) ([]PublisherStatementDTO, int64, error)
}

type PublisherHTTPHandlers struct {
	Publisher            PublisherService
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	ActorUserID          func(*http.Request) (uuid.UUID, bool)
	WriteServiceError    func(http.ResponseWriter, error)
}

func (h *PublisherHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Publisher == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	scoped := []string{"supply:read:scoped"}
	mux.HandleFunc("GET /api/v1/publisher/dashboard", limit(perm(scoped, h.getDashboard)))
	mux.HandleFunc("GET /api/v1/publisher/statements", limit(perm(scoped, h.listStatements)))
}

func (h *PublisherHTTPHandlers) getDashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actorUserID(r)
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	bind, err := h.Publisher.ResolvePublisherBind(r.Context(), userID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	dash, err := h.Publisher.GetPublisherDashboard(r.Context(), bind, from, to)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, dash)
}

func (h *PublisherHTTPHandlers) listStatements(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actorUserID(r)
	if !ok {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	from, to, err := parseReportRange(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	limit, offset := coldpath.ParseAPIPagination(r)
	bind, err := h.Publisher.ResolvePublisherBind(r.Context(), userID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	items, total, err := h.Publisher.ListPublisherStatements(r.Context(), bind, from, to, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "id,amount_micro,created_at,campaign_id,idempotency_hash\n")
		for _, row := range items {
			_, _ = fmt.Fprintf(w, "%d,%d,%s,%s,%s\n", row.ID, row.AmountMicro, row.CreatedAt, row.CampaignID, row.IdempotencyHash)
		}
		return
	}
	httpresponse.JSON(w, http.StatusOK, PublisherStatementListResponse{Items: items, Total: total})
}

func (h *PublisherHTTPHandlers) actorUserID(r *http.Request) (uuid.UUID, bool) {
	if h.ActorUserID != nil {
		return h.ActorUserID(r)
	}
	return uuid.Nil, false
}

func (h *PublisherHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
