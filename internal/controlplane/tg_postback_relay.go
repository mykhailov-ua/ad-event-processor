package controlplane

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	db "github.com/bidshard/ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

const telegramPostbackRelayTimeout = 30 * time.Second

func telegramPostbackRelayContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, telegramPostbackRelayTimeout)
}

func (s *TelegramServiceImpl) relayPostbacks(ctx context.Context, campaignID uuid.UUID, clickID string) {
	if err := ctx.Err(); err != nil {
		return
	}
	if s == nil || s.svc == nil || s.svc.GetPool() == nil {
		return
	}
	q := db.New(s.svc.GetPool())
	postbacks, err := q.ListTelegramPostbacksByCampaign(ctx, ToUUID(campaignID))
	if err != nil || len(postbacks) == 0 {
		return
	}
	client := &http.Client{Timeout: 10 * time.Second}
	for _, p := range postbacks {
		if ctx.Err() != nil {
			return
		}
		url := strings.ReplaceAll(p.PostbackUrl, "{click_id}", clickID)
		url = strings.ReplaceAll(url, "{campaign_id}", campaignID.String())
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
		if err != nil {
			continue
		}
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}
}

func (s *TelegramServiceImpl) recordWebhookStartEvent(
	ctx context.Context,
	campaignID uuid.UUID,
	botID int64,
	token string,
	tgUserID int64,
	chatType string,
	isPremium bool,
) error {
	conn := s.svc.CHWrite()
	if conn == nil {
		return nil
	}
	chCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	userStr := fmt.Sprintf("%d", tgUserID)
	return insertTgEventRaw(chCtx, conn, tgCHInsertRow{
		ClickID:    token,
		CampaignID: campaignID,
		EventType:  "tg_start",
		TgUserID:   userStr,
		StartParam: token,
		ChatType:   chatType,
		IsPremium:  isPremium,
		BotID:      uint64(botID),
	})
}
