package postback

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgtype"
)

const (
	postbackDispatchStatusInFlight  = "IN_FLIGHT"
	postbackDispatchStatusDelivered = "DELIVERED"
	postbackDispatchStatusSent      = "SENT"
	postbackDispatchStatusFailed    = "FAILED"
)

type dispatchSlot int

const (
	dispatchSlotReady dispatchSlot = iota
	dispatchSlotDelivered
	dispatchSlotDuplicate
)

func postbackIdempotencyHash(payload PostbackPayload) string {
	idempotencyStr := fmt.Sprintf("%s|%s|%s", payload.CustomerID, payload.ClickID, payload.EventType)
	hashBytes := sha256.Sum256([]byte(idempotencyStr))
	return hex.EncodeToString(hashBytes[:])
}

func parsePostbackPayload(raw []byte) (PostbackPayload, error) {
	var payload PostbackPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return PostbackPayload{}, fmt.Errorf("failed to unmarshal payload: %w", err)
	}
	return payload, nil
}

type claimedPostbackEvent struct {
	event db.OutboxEvent
	hash  string
	skip  bool
}

func claimPostbackDispatchesInTx(ctx context.Context, q *db.Queries, events []db.OutboxEvent) ([]claimedPostbackEvent, error) {
	claimed := make([]claimedPostbackEvent, 0, len(events))
	for i := range events {
		payload, err := parsePostbackPayload(events[i].Payload)
		if err != nil {
			return nil, err
		}
		hash := postbackIdempotencyHash(payload)
		rows, err := q.InsertPostbackDispatchInFlight(ctx, db.InsertPostbackDispatchInFlightParams{
			IdempotencyHash: hash,
			CampaignID:      pgtype.UUID{Bytes: payload.CampaignID, Valid: true},
			ClickID:         payload.ClickID,
			EventType:       payload.EventType,
		})
		if err != nil {
			return nil, err
		}
		if rows > 0 {
			claimed = append(claimed, claimedPostbackEvent{event: events[i], hash: hash})
			continue
		}
		existing, err := q.GetPostbackDispatch(ctx, hash)
		if err != nil {
			return nil, fmt.Errorf("dispatch claim conflict lookup: %w", err)
		}
		if existing.Status == postbackDispatchStatusSent {
			claimed = append(claimed, claimedPostbackEvent{event: events[i], hash: hash, skip: true})
			continue
		}
		if existing.Status == postbackDispatchStatusDelivered {
			claimed = append(claimed, claimedPostbackEvent{event: events[i], hash: hash})
			continue
		}
		claimed = append(claimed, claimedPostbackEvent{event: events[i], hash: hash})
	}
	return claimed, nil
}
