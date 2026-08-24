package controlplane

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
)

const landerZipUploadMaxBody = landerhost.DefaultMaxZipBytes + (1 << 20)

type landerHostedUploader interface {
	UploadHostedLanderZip(ctx context.Context, landerID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (LanderDTO, error)
}

type landerHostedFileServer interface {
	ServeHostedLanderFile(ctx context.Context, landerID uuid.UUID, relPath string) (io.ReadCloser, string, error)
}

func (h *FlowHTTPHandlers) RegisterHostedLanderRoutes(mux *http.ServeMux) {
	if h == nil || h.Service == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("POST /api/v1/landers/{id}/hosted-upload", limit(perm("campaigns:write", h.uploadHostedLander)))
	mux.HandleFunc("GET /lp/{lander_id}/", h.serveHostedLanderIndex)
	mux.HandleFunc("GET /lp/{lander_id}/{path...}", h.serveHostedLanderPath)
	h.registerHostedEditorRoutes(mux, limit, perm)
}

func (h *FlowHTTPHandlers) uploadHostedLander(w http.ResponseWriter, r *http.Request) {
	id, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid lander id")
		return
	}
	if err := r.ParseMultipartForm(landerZipUploadMaxBody); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid multipart form")
		return
	}
	file, header, err := r.FormFile("zip")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "zip file is required")
		return
	}
	defer file.Close()
	if header.Size > landerhost.DefaultMaxZipBytes {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "zip exceeds size limit")
		return
	}
	ctype := strings.ToLower(header.Header.Get("Content-Type"))
	if ctype != "" && ctype != "application/zip" && ctype != "application/x-zip-compressed" && !strings.HasSuffix(strings.ToLower(header.Filename), ".zip") {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "file must be a zip archive")
		return
	}
	buf, err := io.ReadAll(io.LimitReader(file, landerhost.DefaultMaxZipBytes+1))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read zip")
		return
	}
	if int64(len(buf)) > landerhost.DefaultMaxZipBytes {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "zip exceeds size limit")
		return
	}
	if int64(len(buf)) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "empty zip")
		return
	}
	uploader, ok := h.Service.(landerHostedUploader)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "hosted lander upload is not configured")
		return
	}
	dto, err := uploader.UploadHostedLanderZip(r.Context(), id, bytes.NewReader(buf), int64(len(buf)))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *FlowHTTPHandlers) serveHostedLanderIndex(w http.ResponseWriter, r *http.Request) {
	h.serveHostedLander(w, r, "")
}

func (h *FlowHTTPHandlers) serveHostedLanderPath(w http.ResponseWriter, r *http.Request) {
	h.serveHostedLander(w, r, r.PathValue("path"))
}

func (h *FlowHTTPHandlers) serveHostedLander(w http.ResponseWriter, r *http.Request, relPath string) {
	if h == nil || h.Service == nil {
		http.NotFound(w, r)
		return
	}
	landerID, err := uuid.Parse(strings.TrimSpace(r.PathValue("lander_id")))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	hosted, ok := h.Service.(landerHostedFileServer)
	if !ok {
		http.NotFound(w, r)
		return
	}
	rc, ctype, err := hosted.ServeHostedLanderFile(r.Context(), landerID, relPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	if _, err := io.Copy(w, rc); err != nil {
		return
	}
}
