package telegram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/pkg/coldpath"

	"github.com/jackc/pgx/v5"
)

func (s *Service) HandleOutboxEvent(ctx context.Context, payload []byte) error {
	if s == nil || s.host == nil {
		return errors.New("telegram service unavailable")
	}
	p, err := coldpath.UnmarshalStrict[outbox.TelegramEventPayload](payload)
	if err != nil {
		return err
	}

	var update struct {
		UpdateID int64 `json:"update_id"`
		Message  *struct {
			Chat struct {
				ID   int64  `json:"id"`
				Type string `json:"type"`
			} `json:"chat"`
			Text string `json:"text"`
			From *struct {
				ID        int64 `json:"id"`
				IsPremium bool  `json:"is_premium"`
			} `json:"from"`
		} `json:"message"`
	}
	if err := json.Unmarshal(p.Payload, &update); err != nil {
		return err
	}

	if update.Message == nil || update.Message.Text == "" {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	if !strings.HasPrefix(text, "/start ") {
		return nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(text, "/start"))
	if token == "" {
		return nil
	}

	q := db.New(s.host.Pool())
	deeplink, err := q.GetTelegramDeeplink(ctx, token)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	if time.Now().After(deeplink.ExpiresAt.Time) {
		_ = q.DeleteTelegramDeeplink(ctx, token)
		return nil
	}

	bot, err := q.GetTelegramBotByBotID(ctx, p.BotID)
	if err != nil {
		return err
	}

	isGroup := update.Message.Chat.Type == "group" || update.Message.Chat.Type == "supergroup" || update.Message.Chat.Type == "channel"

	if update.Message.From != nil {
		_ = s.recordWebhookStartEvent(
			ctx,
			FromUUID(deeplink.CampaignID),
			p.BotID,
			token,
			update.Message.From.ID,
			update.Message.Chat.Type,
			update.Message.From.IsPremium,
		)
		relayCtx, relayCancel := telegramPostbackRelayContext(ctx)
		s.relayPostbacks(relayCtx, FromUUID(deeplink.CampaignID), token)
		relayCancel()
	}

	if err := s.limiter.Wait(ctx, update.Message.Chat.ID, isGroup); err != nil {
		return err
	}

	landingURL := bot.MiniAppUrl
	if landingURL == "" {
		landingURL = bot.WebhookUrl
	}
	welcomeMsg := fmt.Sprintf("Welcome! Click here to start the app: %s", landingURL)

	err = s.sendBotMessage(ctx, bot.BotToken, update.Message.Chat.ID, welcomeMsg)
	if err != nil {
		var apiErr *tgAPIError
		if errors.As(err, &apiErr) && apiErr.StatusCode == 429 {
			if apiErr.RetryAfter > 0 {
				s.limiter.BackoffChat(update.Message.Chat.ID, time.Duration(apiErr.RetryAfter)*time.Second)
			}
		}
		return err
	}
	return nil
}
