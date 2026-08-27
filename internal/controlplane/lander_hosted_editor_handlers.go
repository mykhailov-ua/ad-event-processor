package controlplane

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
)

type landerHostedEditorService interface {
	GetHostedEditorState(ctx context.Context, landerID uuid.UUID) (HostedEditorStateDTO, error)
	ReadHostedEditorFile(ctx context.Context, landerID uuid.UUID, relPath string) (HostedEditorFileBodyDTO, error)
	SaveHostedEditorFile(ctx context.Context, landerID uuid.UUID, relPath, content string) (HostedEditorSaveResultDTO, error)
	PublishHostedDraft(ctx context.Context, landerID uuid.UUID, version int) (LanderDTO, error)
	ServeHostedPreviewFile(ctx context.Context, landerID uuid.UUID, version int, relPath, token string) (io.ReadCloser, string, error)
}

func (h *FlowHTTPHandlers) registerHostedEditorRoutes(mux *http.ServeMux, limit func(http.HandlerFunc) http.HandlerFunc, perm func(string, http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/v1/landers/{id}/hosted-editor", limit(perm("campaigns:read", h.getHostedEditorState)))
	mux.HandleFunc("GET /api/v1/landers/{id}/hosted-files/{path...}", limit(perm("campaigns:read", h.readHostedEditorFile)))
	mux.HandleFunc("PUT /api/v1/landers/{id}/hosted-files/{path...}", limit(perm("campaigns:write", h.saveHostedEditorFile)))
	mux.HandleFunc("POST /api/v1/landers/{id}/hosted-publish", limit(perm("campaigns:write", h.publishHostedDraft)))
	mux.HandleFunc("GET /lp-preview/{lander_id}/{path...}", h.serveHostedPreviewPath)
}

func (h *FlowHTTPHandlers) getHostedEditorState(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	svc, ok := h.Service.(landerHostedEditorService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "hosted editor unavailable")
		return
	}
	state, err := svc.GetHostedEditorState(r.Context(), id)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, state)
}

func (h *FlowHTTPHandlers) readHostedEditorFile(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	relPath := hostedEditorFilePath(r.PathValue("path"))
	svc, ok := h.Service.(landerHostedEditorService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "hosted editor unavailable")
		return
	}
	body, err := svc.ReadHostedEditorFile(r.Context(), id, relPath)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, body)
}

func (h *FlowHTTPHandlers) saveHostedEditorFile(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	relPath := hostedEditorFilePath(r.PathValue("path"))
	req, ok := coldpath.DecodeRequestOrBadRequest[HostedEditorFileBodyDTO](w, r, landerhost.DefaultMaxEditorFileBytes)
	if !ok {
		return
	}
	svc, ok := h.Service.(landerHostedEditorService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "hosted editor unavailable")
		return
	}
	result, err := svc.SaveHostedEditorFile(r.Context(), id, relPath, req.Content)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *FlowHTTPHandlers) publishHostedDraft(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[HostedEditorPublishRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	svc, ok := h.Service.(landerHostedEditorService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "hosted editor unavailable")
		return
	}
	dto, err := svc.PublishHostedDraft(r.Context(), id, req.Version)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *FlowHTTPHandlers) serveHostedPreviewPath(w http.ResponseWriter, r *http.Request) {
	h.serveHostedPreview(w, r, r.PathValue("path"))
}

func (h *FlowHTTPHandlers) serveHostedPreview(w http.ResponseWriter, r *http.Request, relPath string) {
	if h == nil || h.Service == nil {
		http.NotFound(w, r)
		return
	}
	landerID, err := uuid.Parse(strings.TrimSpace(r.PathValue("lander_id")))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.NotFound(w, r)
		return
	}
	svc, ok := h.Service.(landerHostedEditorService)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rc, ctype, err := svc.ServeHostedPreviewFile(r.Context(), landerID, 0, relPath, token)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer func() { _ = rc.Close() }()
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "private, no-store")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		return
	}
}

func hostedEditorFilePath(raw string) string {
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return strings.TrimPrefix(raw, "/")
	}
	return strings.TrimPrefix(decoded, "/")
}
