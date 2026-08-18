package controlplane

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type ValidateResult struct {
	Valid     bool   `json:"valid"`
	ClickID   string `json:"click_id,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
	IsPremium bool   `json:"is_premium,omitempty"`
}

type ClickMintResult struct {
	ClickID   string `json:"click_id"`
	ExpiresAt int64  `json:"expires_at"`
}

type BotDTO struct {
	CampaignID  uuid.UUID `json:"campaign_id"`
	BotID       int64     `json:"bot_id"`
	BotToken    string    `json:"bot_token"`
	WebhookURL  string    `json:"webhook_url"`
	MiniAppURL  string    `json:"mini_app_url"`
	SecretToken string    `json:"secret_token"`
	AuthDateTTL int32     `json:"auth_date_ttl"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type DeeplinkDTO struct {
	Token       string     `json:"token"`
	CampaignID  uuid.UUID  `json:"campaign_id"`
	Fbclid      string     `json:"fbclid,omitempty"`
	Ttclid      string     `json:"ttclid,omitempty"`
	UtmSource   string     `json:"utm_source,omitempty"`
	UtmMedium   string     `json:"utm_medium,omitempty"`
	UtmCampaign string     `json:"utm_campaign,omitempty"`
	UtmTerm     string     `json:"utm_term,omitempty"`
	UtmContent  string     `json:"utm_content,omitempty"`
	LandingTS   *time.Time `json:"landing_ts,omitempty"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type PostbackDTO struct {
	ID          uuid.UUID `json:"id"`
	CampaignID  uuid.UUID `json:"campaign_id"`
	PostbackURL string    `json:"postback_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type TelegramService interface {
	ValidateInitData(ctx context.Context, campaignID uuid.UUID, initData string) (ValidateResult, error)
	MintClick(ctx context.Context, campaignID uuid.UUID) (ClickMintResult, error)
	ReceiveWebhook(ctx context.Context, botID int64, secretToken string, body []byte) error
	CreateDeeplink(ctx context.Context, d DeeplinkDTO) (DeeplinkDTO, error)
	GetDeeplink(ctx context.Context, token string) (DeeplinkDTO, error)
	ConfigureBot(ctx context.Context, bot BotDTO) error
	ListBots(ctx context.Context) ([]BotDTO, error)
	GetBot(ctx context.Context, campaignID uuid.UUID) (BotDTO, error)
	CreatePostback(ctx context.Context, p PostbackDTO) error
	UpdatePostback(ctx context.Context, id uuid.UUID, postbackURL string) error
	GetPostback(ctx context.Context, id uuid.UUID) (PostbackDTO, error)
	ListPostbacks(ctx context.Context, campaignID uuid.UUID) ([]PostbackDTO, error)
	DeletePostback(ctx context.Context, id uuid.UUID) error
	TestPostback(ctx context.Context, id uuid.UUID) error
	GetTelegramReport(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)
	GetTelegramSummaryReport(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)
	GetTelegramFunnelReport(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)
	GetTelegramBotsReport(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)
	GetTelegramPremiumReport(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)
	GetTelegramFraudReport(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)
}

type TelegramReportFilter struct {
	CustomerID *uuid.UUID
	CampaignID *uuid.UUID
}

type TelegramHTTPHandlers struct {
	Telegram             TelegramService
	ApplyRateLimit       func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission func([]string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError    func(http.ResponseWriter, error)
}

func (h *TelegramHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Telegram == nil {
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

	mux.HandleFunc("POST /api/v1/telegram/validate", limit(h.validateInitData))
	mux.HandleFunc("POST /api/v1/telegram/clicks", limit(h.clicksMint))
	mux.HandleFunc("POST /api/v1/telegram/webhook/{bot_id}", limit(h.webhook))

	mux.HandleFunc("POST /api/v1/telegram/deeplink-tokens", limit(perm([]string{"campaigns:write"}, h.createDeeplink)))
	mux.HandleFunc("GET /api/v1/telegram/deeplink-tokens/{token}", limit(perm([]string{"campaigns:read"}, h.getDeeplink)))

	mux.HandleFunc("GET /api/v1/telegram/bots", limit(perm([]string{"campaigns:read"}, h.listBots)))
	mux.HandleFunc("GET /api/v1/telegram/bots/{id}", limit(perm([]string{"campaigns:read"}, h.getBot)))
	mux.HandleFunc("PUT /api/v1/telegram/bots/{id}", limit(perm([]string{"campaigns:write"}, h.configureBot)))

	mux.HandleFunc("GET /api/v1/telegram/postbacks", limit(perm([]string{"campaigns:read"}, h.listPostbacks)))
	mux.HandleFunc("POST /api/v1/telegram/postbacks", limit(perm([]string{"campaigns:write"}, h.createPostback)))
	mux.HandleFunc("PUT /api/v1/telegram/postbacks/{id}", limit(perm([]string{"campaigns:write"}, h.updatePostback)))
	mux.HandleFunc("DELETE /api/v1/telegram/postbacks/{id}", limit(perm([]string{"campaigns:write"}, h.deletePostback)))
	mux.HandleFunc("POST /api/v1/telegram/postbacks/{id}/test", limit(perm([]string{"campaigns:write"}, h.testPostback)))

	mux.HandleFunc("GET /api/v1/reports/telegram", limit(perm([]string{"reports:read"}, h.getTelegramReport)))
	mux.HandleFunc("GET /api/v1/reports/telegram/summary", limit(perm([]string{"reports:read"}, h.getTelegramSummaryReport)))
	mux.HandleFunc("GET /api/v1/reports/telegram/funnel", limit(perm([]string{"reports:read"}, h.getTelegramFunnelReport)))
	mux.HandleFunc("GET /api/v1/reports/telegram/bots", limit(perm([]string{"reports:read"}, h.getTelegramBotsReport)))
	mux.HandleFunc("GET /api/v1/reports/telegram/premium", limit(perm([]string{"reports:read"}, h.getTelegramPremiumReport)))
	mux.HandleFunc("GET /api/v1/reports/telegram/fraud", limit(perm([]string{"reports:read"}, h.getTelegramFraudReport)))
	mux.HandleFunc("POST /api/v1/reports/telegram/export", limit(perm([]string{"reports:read"}, h.exportTelegramReport)))
}

type validateReq struct {
	InitData   string `json:"init_data"`
	CampaignID string `json:"campaign_id"`
}

func (h *TelegramHTTPHandlers) validateInitData(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[validateReq](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	campID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	res, err := h.Telegram.ValidateInitData(r.Context(), campID, req.InitData)
	if err != nil {
		if strings.Contains(err.Error(), "expired") || strings.Contains(err.Error(), "validation failed") {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", err.Error())
			return
		}
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

type clicksReq struct {
	CampaignID string `json:"campaign_id"`
}

func (h *TelegramHTTPHandlers) clicksMint(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[clicksReq](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	campID, err := uuid.Parse(req.CampaignID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	res, err := h.Telegram.MintClick(r.Context(), campID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

func (h *TelegramHTTPHandlers) webhook(w http.ResponseWriter, r *http.Request) {
	botIDStr := r.PathValue("bot_id")
	botID, err := strconv.ParseInt(botIDStr, 10, 64)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid bot_id")
		return
	}
	secretToken := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")

	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unable to read request body")
		return
	}

	err = h.Telegram.ReceiveWebhook(r.Context(), botID, secretToken, body)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TelegramHTTPHandlers) createDeeplink(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[DeeplinkDTO](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	res, err := h.Telegram.CreateDeeplink(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

func (h *TelegramHTTPHandlers) getDeeplink(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	res, err := h.Telegram.GetDeeplink(r.Context(), token)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

func (h *TelegramHTTPHandlers) listBots(w http.ResponseWriter, r *http.Request) {
	res, err := h.Telegram.ListBots(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

func (h *TelegramHTTPHandlers) getBot(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	res, err := h.Telegram.GetBot(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

func (h *TelegramHTTPHandlers) configureBot(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[BotDTO](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	req.CampaignID = campaignID
	err = h.Telegram.ConfigureBot(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TelegramHTTPHandlers) listPostbacks(w http.ResponseWriter, r *http.Request) {
	cidStr := r.URL.Query().Get("campaign_id")
	if cidStr == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "campaign_id required")
		return
	}
	campaignID, err := uuid.Parse(cidStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}
	res, err := h.Telegram.ListPostbacks(r.Context(), campaignID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, res)
}

func (h *TelegramHTTPHandlers) createPostback(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[PostbackDTO](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	err := h.Telegram.CreatePostback(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

type updatePostbackReq struct {
	PostbackURL string `json:"postback_url"`
}

func (h *TelegramHTTPHandlers) updatePostback(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid postback id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[updatePostbackReq](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	err = h.Telegram.UpdatePostback(r.Context(), id, req.PostbackURL)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TelegramHTTPHandlers) deletePostback(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid postback id")
		return
	}
	err = h.Telegram.DeletePostback(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TelegramHTTPHandlers) testPostback(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid postback id")
		return
	}
	err = h.Telegram.TestPostback(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *TelegramHTTPHandlers) getTelegramReport(w http.ResponseWriter, r *http.Request) {
	h.writeTelegramReport(w, r, h.Telegram.GetTelegramReport)
}

func (h *TelegramHTTPHandlers) getTelegramSummaryReport(w http.ResponseWriter, r *http.Request) {
	h.writeTelegramReport(w, r, h.Telegram.GetTelegramSummaryReport)
}

func (h *TelegramHTTPHandlers) getTelegramFunnelReport(w http.ResponseWriter, r *http.Request) {
	h.writeTelegramReport(w, r, h.Telegram.GetTelegramFunnelReport)
}

func (h *TelegramHTTPHandlers) getTelegramBotsReport(w http.ResponseWriter, r *http.Request) {
	h.writeTelegramReport(w, r, h.Telegram.GetTelegramBotsReport)
}

func (h *TelegramHTTPHandlers) getTelegramPremiumReport(w http.ResponseWriter, r *http.Request) {
	h.writeTelegramReport(w, r, h.Telegram.GetTelegramPremiumReport)
}

func (h *TelegramHTTPHandlers) getTelegramFraudReport(w http.ResponseWriter, r *http.Request) {
	h.writeTelegramReport(w, r, h.Telegram.GetTelegramFraudReport)
}

func (h *TelegramHTTPHandlers) exportTelegramReport(w http.ResponseWriter, r *http.Request) {
	httpresponse.Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "use POST /api/v1/reports/jobs with report_key=telegram")
}

type telegramReportFn func(ctx context.Context, from, to time.Time, filter TelegramReportFilter) ([]byte, error)

func (h *TelegramHTTPHandlers) writeTelegramReport(w http.ResponseWriter, r *http.Request, fn telegramReportFn) {
	from, to, filter, ok := parseTelegramReportQuery(w, r)
	if !ok {
		return
	}
	res, err := fn(r.Context(), from, to, filter)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(res); err != nil {
		return
	}
}

func parseTelegramReportQuery(w http.ResponseWriter, r *http.Request) (time.Time, time.Time, TelegramReportFilter, bool) {
	q := r.URL.Query()
	fromStr := q.Get("from")
	toStr := q.Get("to")
	from, err1 := time.Parse(time.RFC3339, fromStr)
	to, err2 := time.Parse(time.RFC3339, toStr)
	if err1 != nil || err2 != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid from/to parameters")
		return time.Time{}, time.Time{}, TelegramReportFilter{}, false
	}
	var filter TelegramReportFilter
	if cidStr := q.Get("campaign_id"); cidStr != "" {
		cid, err := uuid.Parse(cidStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return time.Time{}, time.Time{}, TelegramReportFilter{}, false
		}
		filter.CampaignID = &cid
	}
	if custStr := q.Get("customer_id"); custStr != "" {
		custID, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return time.Time{}, time.Time{}, TelegramReportFilter{}, false
		}
		filter.CustomerID = &custID
	}
	return from, to, filter, true
}

func (h *TelegramHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}
