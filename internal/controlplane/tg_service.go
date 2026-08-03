package controlplane

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"espx/internal/controlplane/adminapi"
	db "espx/internal/domain/db"
	"espx/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type tgEventPayload struct {
	TgUserID       string `json:"tg_user_id,omitempty"`
	TgUserIDSha256 string `json:"tg_user_id_sha256,omitempty"`
	StartParam     string `json:"start_param"`
	ChatType       string `json:"chat_type"`
	IsPremium      bool   `json:"is_premium"`
	Motivated      bool   `json:"motivated"`
	WidgetID       string `json:"widget_id"`
	BotID          uint64 `json:"bot_id"`
}

type TelegramServiceImpl struct {
	svc     *Service
	pool    *pgxpool.Pool
	rdbs    []redis.UniversalClient
	limiter *TelegramRateLimiter
}

func NewTelegramService(svc *Service, pool *pgxpool.Pool, rdbs []redis.UniversalClient) *TelegramServiceImpl {
	return &TelegramServiceImpl{
		svc:     svc,
		pool:    pool,
		rdbs:    rdbs,
		limiter: NewTelegramRateLimiter(),
	}
}

func hexSha256(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func FromUUID(pg pgtype.UUID) uuid.UUID {
	if pg.Valid {
		return uuid.UUID(pg.Bytes)
	}
	return uuid.Nil
}

func (s *TelegramServiceImpl) ValidateInitData(ctx context.Context, campaignID uuid.UUID, initData string) (adminapi.ValidateResult, error) {
	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	bot, err := q.GetTelegramBot(pgCtx, ToUUID(campaignID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return adminapi.ValidateResult{Valid: false}, fmt.Errorf("bot not configured for campaign %s", campaignID)
		}
		return adminapi.ValidateResult{Valid: false}, err
	}

	authTTL := int64(bot.AuthDateTtl)
	if authTTL <= 0 {
		authTTL = 300
	}

	params, err := ValidateInitData(initData, bot.BotToken, authTTL)
	if err != nil {
		return adminapi.ValidateResult{Valid: false}, err
	}

	userStr := params["user"]
	var userObj struct {
		ID        int64  `json:"id"`
		IsPremium bool   `json:"is_premium"`
		Username  string `json:"username"`
	}
	if userStr != "" {
		_ = json.Unmarshal([]byte(userStr), &userObj)
	}

	clickID := uuid.New()
	expiresAt := time.Now().Add(15 * time.Minute)

	tgUserIDStr := strconv.FormatInt(userObj.ID, 10)

	meta := tgEventPayload{
		TgUserID:       tgUserIDStr,
		TgUserIDSha256: hexSha256(tgUserIDStr),
		StartParam:     params["start_param"],
		ChatType:       params["chat_type"],
		IsPremium:      userObj.IsPremium,
		Motivated:      false,
		WidgetID:       params["query_id"],
		BotID:          uint64(bot.BotID),
	}
	metaBytes, _ := json.Marshal(meta)

	rdb := s.svc.getRDB(campaignID)
	if rdb == nil {
		return adminapi.ValidateResult{Valid: false}, errors.New("no redis client for campaign")
	}

	redisCtx, cancelR := context.WithTimeout(ctx, 2*time.Second)
	defer cancelR()
	redisKey := fmt.Sprintf("{%s}tg:click:%s", campaignID.String(), clickID.String())
	err = rdb.Set(redisCtx, redisKey, metaBytes, 15*time.Minute).Err()
	if err != nil {
		return adminapi.ValidateResult{Valid: false}, fmt.Errorf("failed to save click to redis: %w", err)
	}

	return adminapi.ValidateResult{
		Valid:     true,
		ClickID:   clickID.String(),
		ExpiresAt: expiresAt.Unix(),
		IsPremium: userObj.IsPremium,
	}, nil
}

func (s *TelegramServiceImpl) MintClick(ctx context.Context, campaignID uuid.UUID) (adminapi.ClickMintResult, error) {
	clickID := uuid.New()
	expiresAt := time.Now().Add(15 * time.Minute)

	meta := tgEventPayload{
		TgUserID:   "",
		StartParam: "",
		BotID:      0,
	}
	metaBytes, _ := json.Marshal(meta)

	rdb := s.svc.getRDB(campaignID)
	if rdb == nil {
		return adminapi.ClickMintResult{}, errors.New("no redis client for campaign")
	}

	redisCtx, cancelR := context.WithTimeout(ctx, 2*time.Second)
	defer cancelR()
	redisKey := fmt.Sprintf("{%s}tg:click:%s", campaignID.String(), clickID.String())
	err := rdb.Set(redisCtx, redisKey, metaBytes, 15*time.Minute).Err()
	if err != nil {
		return adminapi.ClickMintResult{}, fmt.Errorf("failed to save click to redis: %w", err)
	}

	return adminapi.ClickMintResult{
		ClickID:   clickID.String(),
		ExpiresAt: expiresAt.Unix(),
	}, nil
}

func (s *TelegramServiceImpl) ReceiveWebhook(ctx context.Context, botID int64, secretToken string, body []byte) error {
	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	bot, err := q.GetTelegramBotByBotID(pgCtx, botID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("bot %d not configured", botID)
		}
		return err
	}

	if bot.SecretToken != secretToken {
		return errors.New("webhook secret token mismatch")
	}

	var update struct {
		UpdateID int64 `json:"update_id"`
	}
	if err := json.Unmarshal(body, &update); err != nil {
		return fmt.Errorf("invalid update JSON: %w", err)
	}

	err = q.TryClaimTelegramWebhookUpdate(pgCtx, update.UpdateID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique violation
			return nil
		}
		return err
	}

	pBytes, err := coldpath.MarshalOutbox(telegramEventPayload{
		CampaignID: FromUUID(bot.CampaignID),
		BotID:      botID,
		Payload:    body,
	})
	if err != nil {
		return err
	}

	_, err = q.CreateOutboxEvent(pgCtx, db.CreateOutboxEventParams{
		EventType: "TELEGRAM_EVENT",
		Payload:   pBytes,
	})
	return err
}

func (s *TelegramServiceImpl) CreateDeeplink(ctx context.Context, d adminapi.DeeplinkDTO) (adminapi.DeeplinkDTO, error) {
	token, err := GenerateBridgeToken()
	if err != nil {
		return adminapi.DeeplinkDTO{}, err
	}
	d.Token = token
	d.ExpiresAt = time.Now().Add(7 * 24 * time.Hour)

	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	err = q.InsertTelegramDeeplink(pgCtx, db.InsertTelegramDeeplinkParams{
		Token:       token,
		CampaignID:  ToUUID(d.CampaignID),
		Fbclid:      pgtype.Text{String: d.Fbclid, Valid: d.Fbclid != ""},
		Ttclid:      pgtype.Text{String: d.Ttclid, Valid: d.Ttclid != ""},
		UtmSource:   pgtype.Text{String: d.UtmSource, Valid: d.UtmSource != ""},
		UtmMedium:   pgtype.Text{String: d.UtmMedium, Valid: d.UtmMedium != ""},
		UtmCampaign: pgtype.Text{String: d.UtmCampaign, Valid: d.UtmCampaign != ""},
		UtmTerm:     pgtype.Text{String: d.UtmTerm, Valid: d.UtmTerm != ""},
		UtmContent:  pgtype.Text{String: d.UtmContent, Valid: d.UtmContent != ""},
		LandingTs:   pgtype.Timestamptz{Time: time.Now(), Valid: true},
		ExpiresAt:   pgtype.Timestamptz{Time: d.ExpiresAt, Valid: true},
	})
	if err != nil {
		return adminapi.DeeplinkDTO{}, err
	}

	rdb := s.svc.getRDB(d.CampaignID)
	if rdb != nil {
		redisCtx, cancelR := context.WithTimeout(ctx, 2*time.Second)
		defer cancelR()
		redisKey := "tg:deeplink:" + token
		dBytes, _ := json.Marshal(d)
		set, err := rdb.SetNX(redisCtx, redisKey, dBytes, 7*24*time.Hour).Result()
		if err == nil && !set {
			if cached, getErr := rdb.Get(redisCtx, redisKey).Bytes(); getErr == nil {
				_ = json.Unmarshal(cached, &d)
			}
		}
	}

	return d, nil
}

func (s *TelegramServiceImpl) GetDeeplink(ctx context.Context, token string) (adminapi.DeeplinkDTO, error) {
	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	row, err := q.GetTelegramDeeplink(pgCtx, token)
	if err != nil {
		return adminapi.DeeplinkDTO{}, err
	}
	var landingTS *time.Time
	if row.LandingTs.Valid {
		landingTS = &row.LandingTs.Time
	}
	return adminapi.DeeplinkDTO{
		Token:       row.Token,
		CampaignID:  FromUUID(row.CampaignID),
		Fbclid:      row.Fbclid.String,
		Ttclid:      row.Ttclid.String,
		UtmSource:   row.UtmSource.String,
		UtmMedium:   row.UtmMedium.String,
		UtmCampaign: row.UtmCampaign.String,
		UtmTerm:     row.UtmTerm.String,
		UtmContent:  row.UtmContent.String,
		LandingTS:   landingTS,
		ExpiresAt:   row.ExpiresAt.Time,
	}, nil
}

func (s *TelegramServiceImpl) ConfigureBot(ctx context.Context, bot adminapi.BotDTO) error {
	q := db.New(s.pool)
	existing, err := q.GetTelegramBot(ctx, ToUUID(bot.CampaignID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return q.InsertTelegramBot(ctx, db.InsertTelegramBotParams{
				CampaignID:  ToUUID(bot.CampaignID),
				BotID:       bot.BotID,
				BotToken:    bot.BotToken,
				WebhookUrl:  bot.WebhookURL,
				MiniAppUrl:  bot.MiniAppURL,
				SecretToken: bot.SecretToken,
				AuthDateTtl: bot.AuthDateTTL,
			})
		}
		return err
	}
	_ = existing
	return q.UpdateTelegramBot(ctx, db.UpdateTelegramBotParams{
		CampaignID:  ToUUID(bot.CampaignID),
		BotID:       bot.BotID,
		BotToken:    bot.BotToken,
		WebhookUrl:  bot.WebhookURL,
		MiniAppUrl:  bot.MiniAppURL,
		SecretToken: bot.SecretToken,
		AuthDateTtl: bot.AuthDateTTL,
	})
}

func (s *TelegramServiceImpl) ListBots(ctx context.Context) ([]adminapi.BotDTO, error) {
	q := db.New(s.pool)
	rows, err := q.ListTelegramBots(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]adminapi.BotDTO, len(rows))
	for i, row := range rows {
		res[i] = adminapi.BotDTO{
			CampaignID:  FromUUID(row.CampaignID),
			BotID:       row.BotID,
			BotToken:    row.BotToken,
			WebhookURL:  row.WebhookUrl,
			MiniAppURL:  row.MiniAppUrl,
			SecretToken: row.SecretToken,
			AuthDateTTL: row.AuthDateTtl,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return res, nil
}

func (s *TelegramServiceImpl) GetBot(ctx context.Context, campaignID uuid.UUID) (adminapi.BotDTO, error) {
	q := db.New(s.pool)
	row, err := q.GetTelegramBot(ctx, ToUUID(campaignID))
	if err != nil {
		return adminapi.BotDTO{}, err
	}
	return adminapi.BotDTO{
		CampaignID:  FromUUID(row.CampaignID),
		BotID:       row.BotID,
		BotToken:    row.BotToken,
		WebhookURL:  row.WebhookUrl,
		MiniAppURL:  row.MiniAppUrl,
		SecretToken: row.SecretToken,
		AuthDateTTL: row.AuthDateTtl,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (s *TelegramServiceImpl) CreatePostback(ctx context.Context, p adminapi.PostbackDTO) error {
	q := db.New(s.pool)
	return q.InsertTelegramPostback(ctx, db.InsertTelegramPostbackParams{
		ID:          ToUUID(p.ID),
		CampaignID:  ToUUID(p.CampaignID),
		PostbackUrl: p.PostbackURL,
	})
}

func (s *TelegramServiceImpl) UpdatePostback(ctx context.Context, id uuid.UUID, postbackURL string) error {
	q := db.New(s.pool)
	return q.UpdateTelegramPostback(ctx, db.UpdateTelegramPostbackParams{
		ID:          ToUUID(id),
		PostbackUrl: postbackURL,
	})
}

func (s *TelegramServiceImpl) GetPostback(ctx context.Context, id uuid.UUID) (adminapi.PostbackDTO, error) {
	q := db.New(s.pool)
	row, err := q.GetTelegramPostback(ctx, ToUUID(id))
	if err != nil {
		return adminapi.PostbackDTO{}, err
	}
	return adminapi.PostbackDTO{
		ID:          FromUUID(row.ID),
		CampaignID:  FromUUID(row.CampaignID),
		PostbackURL: row.PostbackUrl,
		CreatedAt:   row.CreatedAt.Time,
		UpdatedAt:   row.UpdatedAt.Time,
	}, nil
}

func (s *TelegramServiceImpl) ListPostbacks(ctx context.Context, campaignID uuid.UUID) ([]adminapi.PostbackDTO, error) {
	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	rows, err := q.ListTelegramPostbacksByCampaign(pgCtx, ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	res := make([]adminapi.PostbackDTO, len(rows))
	for i, row := range rows {
		res[i] = adminapi.PostbackDTO{
			ID:          FromUUID(row.ID),
			CampaignID:  FromUUID(row.CampaignID),
			PostbackURL: row.PostbackUrl,
			CreatedAt:   row.CreatedAt.Time,
			UpdatedAt:   row.UpdatedAt.Time,
		}
	}
	return res, nil
}

func (s *TelegramServiceImpl) DeletePostback(ctx context.Context, id uuid.UUID) error {
	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	return q.DeleteTelegramPostback(pgCtx, ToUUID(id))
}

type tgAPIError struct {
	StatusCode int
	RetryAfter int
}

func (e *tgAPIError) Error() string {
	return fmt.Sprintf("telegram bot api error: status=%d retry_after=%d", e.StatusCode, e.RetryAfter)
}

func (s *TelegramServiceImpl) sendBotMessage(ctx context.Context, botToken string, chatID int64, text string) error {
	if s.limiter != nil {
		if err := s.limiter.Wait(ctx, chatID, false); err != nil {
			return err
		}
	}

	client := &http.Client{Timeout: 5 * time.Second}
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)

	bodyData := map[string]interface{}{
		"chat_id": chatID,
		"text":    text,
	}
	bodyBytes, _ := json.Marshal(bodyData)

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errData struct {
			Parameters struct {
				RetryAfter int `json:"retry_after"`
			} `json:"parameters"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errData)

		if resp.StatusCode == http.StatusTooManyRequests && s.limiter != nil {
			if errData.Parameters.RetryAfter > 0 {
				s.limiter.BackoffChat(chatID, time.Duration(errData.Parameters.RetryAfter)*time.Second)
			}
		}

		return &tgAPIError{
			StatusCode: resp.StatusCode,
			RetryAfter: errData.Parameters.RetryAfter,
		}
	}
	return nil
}

func (s *TelegramServiceImpl) TestPostback(ctx context.Context, id uuid.UUID) error {
	pgCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	q := db.New(s.pool)
	p, err := q.GetTelegramPostback(pgCtx, ToUUID(id))
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	testURL := strings.ReplaceAll(p.PostbackUrl, "{click_id}", "test_click_id")
	testURL = strings.ReplaceAll(testURL, "{campaign_id}", FromUUID(p.CampaignID).String())
	req, err := http.NewRequestWithContext(ctx, "GET", testURL, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("postback url returned status code %d", resp.StatusCode)
	}
	return nil
}
