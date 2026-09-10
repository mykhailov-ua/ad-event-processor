package main

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"ad-event-processor/internal/licenseissue"
	"ad-event-processor/internal/trialregistry"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

type handler struct {
	svc   *licenseissue.Service
	token string
}

func newHandler(svc *licenseissue.Service, token string) *handler {
	return &handler{svc: svc, token: token}
}

func (h *handler) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /api/v1/vendor/catalog", h.auth(h.catalog))
	mux.HandleFunc("GET /api/v1/vendor/plans/buttons", h.auth(h.planButtons))
	mux.HandleFunc("POST /api/v1/vendor/offer-acceptances", h.auth(h.acceptOffer))
	mux.HandleFunc("GET /api/v1/vendor/trial-requests", h.auth(h.listPending))
	mux.HandleFunc("POST /api/v1/vendor/trial-requests", h.auth(h.createPending))
	mux.HandleFunc("GET /api/v1/vendor/trial-requests/{id}", h.auth(h.getPending))
	mux.HandleFunc("POST /api/v1/vendor/trial-requests/{id}/issue", h.auth(h.issuePending))
	mux.HandleFunc("POST /api/v1/vendor/trial-requests/{id}/reject", h.auth(h.rejectPending))
	mux.HandleFunc("POST /api/v1/vendor/licenses", h.auth(h.issueLicense))
	return mux
}

func (h *handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.authorized(r) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "missing or invalid bearer token")
			return
		}
		next(w, r)
	}
}

func (h *handler) authorized(r *http.Request) bool {
	got := strings.TrimSpace(r.Header.Get("Authorization"))
	if got == "" {
		return false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(got, prefix) {
		return false
	}
	return strings.TrimSpace(got[len(prefix):]) == h.token
}

func (h *handler) health(w http.ResponseWriter, _ *http.Request) {
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) catalog(w http.ResponseWriter, _ *http.Request) {
	items, err := h.svc.Catalog()
	if err != nil {
		writeIssueErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{"skus": items})
}

func (h *handler) planButtons(w http.ResponseWriter, _ *http.Request) {
	rows, err := h.svc.PlanButtons()
	if err != nil {
		writeIssueErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, rows)
}

type createPendingRequest struct {
	TelegramID       string `json:"telegram_id"`
	TelegramUsername string `json:"telegram_username"`
	Notes            string `json:"notes"`
	OfferVersion     string `json:"offer_version"`
}

type createPendingResponse struct {
	Request       trialregistry.PendingRequest `json:"request"`
	UserMessage   string                       `json:"user_message"`
	AlreadyQueued bool                         `json:"already_queued"`
}

type acceptOfferRequest struct {
	TelegramID   string `json:"telegram_id"`
	OfferVersion string `json:"offer_version"`
	Source       string `json:"source"`
}

func (h *handler) acceptOffer(w http.ResponseWriter, r *http.Request) {
	body, ok := coldpath.DecodeRequestOrBadRequest[acceptOfferRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	offerVersion := strings.TrimSpace(body.OfferVersion)
	if offerVersion == "" {
		offerVersion = trialregistry.CurrentOfferVersion()
	}
	source := strings.TrimSpace(body.Source)
	if source == "" {
		source = trialregistry.AcceptSourceVendorAPI
	}
	if err := h.svc.Registry().AcceptOffer(body.TelegramID, offerVersion, source); err != nil {
		writeTrialErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{
		"status":        "accepted",
		"telegram_id":   strings.TrimSpace(body.TelegramID),
		"offer_version": offerVersion,
		"source":        source,
	})
}

func (h *handler) createPending(w http.ResponseWriter, r *http.Request) {
	body, ok := coldpath.DecodeRequestOrBadRequest[createPendingRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	reg := h.svc.Registry()
	openBefore := ""
	pending, _ := reg.ListPending()
	for _, item := range pending {
		if item.TelegramID == strings.TrimSpace(body.TelegramID) {
			openBefore = item.ID
			break
		}
	}
	offerVersion := strings.TrimSpace(body.OfferVersion)
	if offerVersion == "" {
		offerVersion = trialregistry.CurrentOfferVersion()
	}
	if offerVersion != trialregistry.CurrentOfferVersion() {
		httpresponse.Error(w, http.StatusBadRequest, "OFFER_VERSION_MISMATCH", "offer_version mismatch")
		return
	}
	accepted, err := reg.HasOfferAcceptance(strings.TrimSpace(body.TelegramID), offerVersion)
	if err != nil {
		writeTrialErr(w, err)
		return
	}
	if !accepted {
		httpresponse.Error(w, http.StatusBadRequest, "OFFER_NOT_ACCEPTED", "record offer acceptance before enqueueing pilot")
		return
	}
	req, err := reg.EnqueuePending(trialregistry.EnqueuePendingInput{
		TelegramID:       body.TelegramID,
		TelegramUsername: body.TelegramUsername,
		Notes:            body.Notes,
		OfferVersion:     offerVersion,
	})
	if err != nil {
		writeTrialErr(w, err)
		return
	}
	resp := createPendingResponse{
		Request: req,
		UserMessage: fmt.Sprintf(
			"Pilot request queued (id=%s). A vendor operator will approve after you send your HWID.",
			req.ID,
		),
		AlreadyQueued: openBefore != "" && openBefore == req.ID,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (h *handler) listPending(w http.ResponseWriter, _ *http.Request) {
	items, err := h.svc.Registry().ListPending()
	if err != nil {
		writeIssueErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, licenseissue.PendingListView(items))
}

func (h *handler) getPending(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	req, err := h.svc.Registry().GetPending(id)
	if err != nil {
		writeTrialErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{"request": req})
}

type issuePendingRequest struct {
	HWIDV2       string `json:"hwid_v2"`
	DeploymentID string `json:"deployment_id"`
	Operator     string `json:"operator"`
	Customer     string `json:"customer"`
}

type issueLicenseRequest struct {
	SKUCode          string `json:"sku_code"`
	Customer         string `json:"customer"`
	DeploymentID     string `json:"deployment_id"`
	HWIDV2           string `json:"hwid_v2"`
	TelegramID       string `json:"telegram_id"`
	USDTTx           string `json:"usdt_tx"`
	ValidDays        int    `json:"valid_days"`
	Operator         string `json:"operator"`
	Force            bool   `json:"force"`
	ForceReason      string `json:"force_reason"`
	ApprovePendingID string `json:"approve_pending_id"`
}

type licenseIssueResponse struct {
	DeploymentID string                        `json:"deployment_id"`
	KeyID        string                        `json:"key_id"`
	ValidUntil   string                        `json:"valid_until"`
	LicenseKey   string                        `json:"license_key"`
	LicenseJWT   string                        `json:"license_jwt"`
	Telegram     licenseissue.TelegramDelivery `json:"telegram"`
}

func (h *handler) issuePending(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	body, ok := coldpath.DecodeRequestOrBadRequest[issuePendingRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	res, err := h.svc.Issue(licenseissue.IssueRequest{
		SKUCode:          "pilot",
		Customer:         body.Customer,
		DeploymentID:     body.DeploymentID,
		HWIDV2:           body.HWIDV2,
		Operator:         body.Operator,
		ApprovePendingID: id,
	})
	if err != nil {
		writeIssueErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, toLicenseResponse(res, "pilot"))
}

type rejectPendingRequest struct {
	Reason string `json:"reason"`
}

func (h *handler) rejectPending(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	body, ok := coldpath.DecodeRequestOrBadRequest[rejectPendingRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	if err := h.svc.Registry().RejectPending(id, body.Reason); err != nil {
		writeTrialErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{
		"status":  "rejected",
		"id":      id,
		"message": "Pilot request rejected.",
	})
}

func (h *handler) issueLicense(w http.ResponseWriter, r *http.Request) {
	body, ok := coldpath.DecodeRequestOrBadRequest[issueLicenseRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	res, err := h.svc.Issue(licenseissue.IssueRequest{
		SKUCode:          body.SKUCode,
		Customer:         body.Customer,
		DeploymentID:     body.DeploymentID,
		HWIDV2:           body.HWIDV2,
		TelegramID:       body.TelegramID,
		USDTTx:           body.USDTTx,
		ValidDays:        body.ValidDays,
		Operator:         body.Operator,
		Force:            body.Force,
		ForceReason:      body.ForceReason,
		ApprovePendingID: body.ApprovePendingID,
	})
	if err != nil {
		writeIssueErr(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, toLicenseResponse(res, body.SKUCode))
}

func toLicenseResponse(res licenseissue.IssueResult, sku string) licenseIssueResponse {
	sku = strings.TrimSpace(sku)
	if sku == "" {
		sku = "pilot"
	}
	return licenseIssueResponse{
		DeploymentID: res.DeploymentID,
		KeyID:        res.KeyID,
		ValidUntil:   res.ValidUntil.UTC().Format(time.RFC3339),
		LicenseKey:   res.LicenseKey,
		LicenseJWT:   res.Token,
		Telegram:     licenseissue.IssueTelegramDelivery(res, sku),
	}
}

func writeIssueErr(w http.ResponseWriter, err error) {
	status, code := licenseissue.MapIssueError(err)
	msg := err.Error()
	if status >= 500 {
		msg = "internal error"
	}
	httpresponse.Error(w, status, code, msg)
}

func writeTrialErr(w http.ResponseWriter, err error) {
	switch {
	case err == trialregistry.ErrTrialTelegramUsed:
		httpresponse.Error(w, http.StatusConflict, "TRIAL_TELEGRAM_USED", err.Error())
	case err == trialregistry.ErrPendingNotFound:
		httpresponse.Error(w, http.StatusNotFound, "PENDING_NOT_FOUND", err.Error())
	case err == trialregistry.ErrPendingNotOpen:
		httpresponse.Error(w, http.StatusConflict, "PENDING_NOT_OPEN", err.Error())
	case err == trialregistry.ErrOfferNotAccepted:
		httpresponse.Error(w, http.StatusBadRequest, "OFFER_NOT_ACCEPTED", err.Error())
	case err == trialregistry.ErrOfferVersionMismatch:
		httpresponse.Error(w, http.StatusBadRequest, "OFFER_VERSION_MISMATCH", err.Error())
	default:
		if strings.Contains(err.Error(), "required") {
			httpresponse.Error(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
	}
}
