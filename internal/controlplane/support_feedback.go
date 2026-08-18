package controlplane

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"
	"github.com/bidshard/ad-event-processor/pkg/supportbundle"

	"github.com/google/uuid"
)

type SupportFeedbackMeta struct {
	DeploymentID  string `json:"deployment_id"`
	BinaryVersion string `json:"binary_version"`
}

type SupportFeedbackRecorder interface {
	SupportFeedbackMeta(ctx context.Context) (SupportFeedbackMeta, error)
	RecordSupportFeedback(ctx context.Context, in SupportFeedbackRecord) (uuid.UUID, error)
}

type SupportFeedbackRecord struct {
	Type          string
	ContactEmail  string
	Message       string
	AttachBundle  bool
	BundleGzip    []byte
	SubmitterID   uuid.UUID
	DeploymentID  string
	BinaryVersion string
}

type SupportHTTPHandlers struct {
	Feedback          SupportFeedbackRecorder
	SupportBundle     SupportBundleWriter
	ApplyRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequireAuth       func(http.HandlerFunc) http.HandlerFunc
	WriteServiceError func(http.ResponseWriter, error)
}

func (h *SupportHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Feedback == nil {
		return
	}
	limit := h.ApplyRateLimit
	auth := h.RequireAuth
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if auth == nil {
		auth = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/support/feedback/meta", limit(auth(h.getSupportFeedbackMeta)))
	mux.HandleFunc("POST /api/v1/support/feedback", limit(auth(h.postSupportFeedback)))
}

type supportFeedbackRequest struct {
	Type         string `json:"type"`
	ContactEmail string `json:"contact_email"`
	Message      string `json:"message"`
	AttachBundle bool   `json:"attach_bundle"`
}

type supportFeedbackResponse struct {
	ID string `json:"id"`
}

func (h *SupportHTTPHandlers) getSupportFeedbackMeta(w http.ResponseWriter, r *http.Request) {
	meta, err := h.Feedback.SupportFeedbackMeta(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, meta)
}

func (h *SupportHTTPHandlers) postSupportFeedback(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[supportFeedbackRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	meta, err := h.Feedback.SupportFeedbackMeta(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	record := SupportFeedbackRecord{
		Type:          req.Type,
		ContactEmail:  req.ContactEmail,
		Message:       req.Message,
		AttachBundle:  req.AttachBundle,
		DeploymentID:  meta.DeploymentID,
		BinaryVersion: meta.BinaryVersion,
	}
	if req.AttachBundle {
		if h.SupportBundle == nil {
			httpresponse.Error(w, http.StatusServiceUnavailable, "BUNDLE_UNAVAILABLE", "support bundle not configured")
			return
		}
		bundle, bundleErr := h.generateFeedbackBundle(r.Context())
		if bundleErr != nil {
			h.writeServiceError(w, bundleErr)
			return
		}
		record.BundleGzip = bundle
	}

	id, err := h.Feedback.RecordSupportFeedback(r.Context(), record)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, supportFeedbackResponse{ID: id.String()})
}

func (h *SupportHTTPHandlers) generateFeedbackBundle(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, supportbundle.DefaultTimeout)
	defer cancel()
	var buf bytes.Buffer
	if err := h.SupportBundle.WriteSupportBundle(ctx, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (h *SupportHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}

var bundleForbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)https?://`),
	regexp.MustCompile(`sk_live`),
	regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`),
	regexp.MustCompile(`lk_secret`),
}

func validateFeedbackBundleRedaction(bundle []byte) error {
	if len(bundle) < 2 || bundle[0] != 0x1f || bundle[1] != 0x8b {
		return io.ErrUnexpectedEOF
	}
	for _, re := range bundleForbiddenPatterns {
		if re.Match(bundle) {
			return io.ErrNoProgress
		}
	}
	return nil
}
