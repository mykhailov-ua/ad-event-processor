package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
)

type tgCHInsertRow struct {
	ClickID    string
	CampaignID uuid.UUID
	EventType  string
	TgUserID   string
	StartParam string
	ChatType   string
	IsPremium  bool
	Motivated  bool
	WidgetID   string
	BotID      uint64
	Payload    []byte
	CreatedAt  time.Time
}

func insertTgEventRaw(ctx context.Context, conn driver.Conn, row tgCHInsertRow) error {
	if conn == nil {
		return nil
	}
	if row.CreatedAt.IsZero() {
		row.CreatedAt = time.Now().UTC()
	}
	payload := row.Payload
	if len(payload) == 0 {
		payload, _ = json.Marshal(map[string]any{
			"tg_user_id_sha256": hexSha256(row.TgUserID),
			"start_param":       row.StartParam,
			"chat_type":         row.ChatType,
			"bot_id":            row.BotID,
			"source":            "telegram",
		})
	}
	var premium, motivated uint8
	if row.IsPremium {
		premium = 1
	}
	if row.Motivated {
		motivated = 1
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO tg_events_raw")
	if err != nil {
		return fmt.Errorf("prepare tg_events_raw batch: %w", err)
	}
	if err := batch.Append(
		row.ClickID,
		row.CampaignID,
		row.TgUserID,
		row.StartParam,
		row.ChatType,
		premium,
		motivated,
		row.WidgetID,
		row.BotID,
		string(payload),
		row.CreatedAt,
		row.EventType,
	); err != nil {
		return fmt.Errorf("append tg_events_raw: %w", err)
	}
	if err := batch.Send(); err != nil {
		return fmt.Errorf("send tg_events_raw: %w", err)
	}
	return nil
}
