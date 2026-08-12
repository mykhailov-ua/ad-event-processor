package controlplane

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/ingestion"
	"github.com/bidshard/ad-event-processor/internal/ingestion/pb"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/proto"
)

var ErrDLQEntryNotFound = errors.New("dlq entry not found")

const dlqRetryIdempotencyTTL = 24 * time.Hour

func eventsStreamName(cfg *config.Config) string {
	if cfg != nil && cfg.RedisStreamName != "" {
		return cfg.RedisStreamName
	}
	return "ad:events:stream"
}

func dlqStreamName(cfg *config.Config) string {
	stream := eventsStreamName(cfg)
	const suffix = ":stream"
	if strings.HasSuffix(stream, suffix) {
		return stream[:len(stream)-len(suffix)] + ":dlq"
	}
	if stream == "" {
		return "ad:events:dlq"
	}
	return stream + ":dlq"
}

func dlqRouteID(shardID int, entryID string) string {
	return fmt.Sprintf("shard-%d-%s", shardID, entryID)
}

func (r *opsReader) listDLQEntries(ctx context.Context, cursor string, limit int) (adminapi.FanOutResult[adminapi.DLQEntryDTO], error) {
	if r == nil || r.svc == nil {
		return adminapi.FanOutResult[adminapi.DLQEntryDTO]{}, fmt.Errorf("ops reader not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	rdbs := r.svc.rdbs
	if len(rdbs) == 0 {
		return adminapi.FanOutResult[adminapi.DLQEntryDTO]{}, fmt.Errorf("redis not configured")
	}

	sourceCursors, err := adminapi.DecodeFanOutCursor(cursor)
	if err != nil {
		return adminapi.FanOutResult[adminapi.DLQEntryDTO]{}, errInvalidQuery("invalid cursor")
	}

	dlqStream := dlqStreamName(r.svc.cfg)
	eventStream := eventsStreamName(r.svc.cfg)

	type shardBatch struct {
		shardID int
		items   []adminapi.DLQEntryDTO
		next    string
		err     error
	}

	batches := make([]shardBatch, 0, len(rdbs))
	for shardID, rdb := range rdbs {
		if rdb == nil {
			batches = append(batches, shardBatch{
				shardID: shardID,
				err:     fmt.Errorf("shard %d unavailable", shardID),
			})
			continue
		}
		start := sourceCursors[strconv.Itoa(shardID)]
		items, next, readErr := readDLQShardPage(ctx, rdb, shardID, dlqStream, eventStream, start, limit)
		batches = append(batches, shardBatch{
			shardID: shardID,
			items:   items,
			next:    next,
			err:     readErr,
		})
	}

	var (
		out      adminapi.FanOutResult[adminapi.DLQEntryDTO]
		merged   []adminapi.DLQEntryDTO
		nextMap  = make(map[string]string, len(batches))
		okShards int
	)
	for _, batch := range batches {
		if batch.err != nil {
			out.Errors = append(out.Errors, adminapi.FanOutSourceError{
				Source: fmt.Sprintf("shard-%d", batch.shardID),
				Code:   "SOURCE_UNAVAILABLE",
			})
			continue
		}
		okShards++
		if len(batch.items) > 0 {
			merged = append(merged, batch.items...)
		}
		if batch.next != "" {
			nextMap[strconv.Itoa(batch.shardID)] = batch.next
		}
	}

	sort.Slice(merged, func(i, j int) bool {
		return merged[i].FailedAt > merged[j].FailedAt
	})
	if len(merged) > limit {
		merged = merged[:limit]
	}
	out.Items = merged

	if len(merged) == limit {
		for _, item := range merged {
			nextMap[strconv.Itoa(item.ShardID)] = item.EntryID
		}
	}
	if encoded, encErr := adminapi.EncodeFanOutCursor(nextMap); encErr == nil {
		out.NextCursor = encoded
	}
	if len(out.Errors) > 0 && okShards > 0 {
		out.Partial = true
	}
	return out, nil
}

func readDLQShardPage(
	ctx context.Context,
	rdb redis.UniversalClient,
	shardID int,
	dlqStream, eventStream, start string,
	limit int,
) ([]adminapi.DLQEntryDTO, string, error) {
	rangeStart := "-"
	if start != "" {
		rangeStart = "(" + start
	}
	msgs, err := rdb.XRangeN(ctx, dlqStream, rangeStart, "+", int64(limit)).Result()
	if err != nil {
		return nil, "", err
	}

	items := make([]adminapi.DLQEntryDTO, 0, len(msgs))
	var lastID string
	for _, msg := range msgs {
		dto, parseErr := dlqMessageToDTO(shardID, dlqStream, eventStream, msg)
		if parseErr != nil {
			continue
		}
		items = append(items, dto)
		lastID = msg.ID
	}
	return items, lastID, nil
}

func dlqMessageToDTO(shardID int, dlqStream, eventStream string, msg redis.XMessage) (adminapi.DLQEntryDTO, error) {
	pbDLQ := &pb.AdDLQEvent{}
	if raw, ok := msg.Values["d"].(string); ok {
		if err := proto.Unmarshal(ingestion.UnsafeBytes(raw), pbDLQ); err != nil {
			return adminapi.DLQEntryDTO{}, err
		}
	} else {
		pbStream := &pb.AdStreamEvent{}
		if v, ok := msg.Values["click_id"].(string); ok {
			pbStream.ClickId = ingestion.UnsafeBytes(v)
		}
		if v, ok := msg.Values["campaign_id"].(string); ok {
			if u, err := uuid.Parse(v); err == nil {
				pbStream.CampaignId = u[:]
			} else {
				pbStream.CampaignId = ingestion.UnsafeBytes(v)
			}
		}
		if v, ok := msg.Values["type"].(string); ok {
			pbStream.EventType = ingestion.UnsafeBytes(v)
		}
		if v, ok := msg.Values["error"].(string); ok {
			pbDLQ.Error = ingestion.UnsafeBytes(v)
		}
		pbDLQ.OriginalEvent = pbStream
	}
	if pbDLQ.OriginalEvent == nil {
		pbDLQ.OriginalEvent = &pb.AdStreamEvent{}
	}

	entryID := msg.ID
	if len(pbDLQ.OriginalId) > 0 {
		entryID = string(pbDLQ.OriginalId)
	}

	failedAt := time.Unix(pbDLQ.FailedAtUnix, 0).UTC()
	if pbDLQ.FailedAtUnix == 0 {
		if ts, ok := parseRedisStreamIDMillis(msg.ID); ok {
			failedAt = time.UnixMilli(ts).UTC()
		} else {
			failedAt = time.Now().UTC()
		}
	}

	campaignID := ""
	if len(pbDLQ.OriginalEvent.CampaignId) == 16 {
		campaignID = uuid.UUID(pbDLQ.OriginalEvent.CampaignId).String()
	} else if len(pbDLQ.OriginalEvent.CampaignId) > 0 {
		campaignID = string(pbDLQ.OriginalEvent.CampaignId)
	}

	return adminapi.DLQEntryDTO{
		ID:         dlqRouteID(shardID, msg.ID),
		ShardID:    shardID,
		StreamID:   eventStream,
		EntryID:    entryID,
		CampaignID: campaignID,
		EventType:  string(pbDLQ.OriginalEvent.EventType),
		Error:      string(pbDLQ.Error),
		FailedAt:   failedAt.Format(time.RFC3339),
		RetryCount: pbDLQ.RetryCount,
		WorkerID:   string(pbDLQ.WorkerId),
	}, nil
}

func parseRedisStreamIDMillis(id string) (int64, bool) {
	dash := strings.IndexByte(id, '-')
	if dash <= 0 {
		return 0, false
	}
	ms, err := strconv.ParseInt(id[:dash], 10, 64)
	if err != nil {
		return 0, false
	}
	return ms, true
}

func (r *opsReader) enqueueDLQRetry(ctx context.Context, payload adminapi.DLQRetryPayload, idempotencyKey string) error {
	if r == nil || r.svc == nil {
		return fmt.Errorf("ops reader not configured")
	}
	if payload.EntryID == "" {
		return errInvalidQuery("entry_id required")
	}
	rdbs := r.svc.rdbs
	if len(rdbs) == 0 {
		return fmt.Errorf("redis not configured")
	}
	shardID := payload.ShardID
	if shardID < 0 || shardID >= len(rdbs) || rdbs[shardID] == nil {
		return errInvalidQuery("invalid shard_id")
	}
	rdb := rdbs[shardID]

	dlqStream := payload.Stream
	if dlqStream == "" {
		dlqStream = dlqStreamName(r.svc.cfg)
	}
	targetStream := eventsStreamName(r.svc.cfg)

	dlqMsgID := payload.EntryID
	if payload.DLQID != "" {
		if parsed := adminapiParseDLQEntryIDFromRoute(payload.DLQID); parsed != "" {
			dlqMsgID = parsed
		}
	}

	if idempotencyKey != "" {
		idemKey := fmt.Sprintf("ops:dlq-retry:%s:%s", payload.DLQID, idempotencyKey)
		ok, err := rdb.SetNX(ctx, idemKey, "1", dlqRetryIdempotencyTTL).Result()
		if err != nil {
			return fmt.Errorf("dlq retry idempotency: %w", err)
		}
		if !ok {
			return nil
		}
	}

	msgs, err := rdb.XRange(ctx, dlqStream, dlqMsgID, dlqMsgID).Result()
	if err != nil {
		return err
	}
	if len(msgs) == 0 {
		return ErrDLQEntryNotFound
	}

	values := make(map[string]interface{})
	msg := msgs[0]
	if raw, ok := msg.Values["d"].(string); ok {
		pbDLQ := &pb.AdDLQEvent{}
		if unmarshalErr := proto.Unmarshal(ingestion.UnsafeBytes(raw), pbDLQ); unmarshalErr != nil {
			return fmt.Errorf("unmarshal dlq event: %w", unmarshalErr)
		}
		if pbDLQ.OriginalEvent == nil {
			return fmt.Errorf("dlq entry has no original event")
		}
		data, marshalErr := proto.Marshal(pbDLQ.OriginalEvent)
		if marshalErr != nil {
			return fmt.Errorf("marshal original event: %w", marshalErr)
		}
		values["d"] = ingestion.UnsafeString(data)
	} else {
		for k, v := range msg.Values {
			switch k {
			case "error", "original_id", "failed_at", "failed_at_unix", "service", "worker_id", "retry_count":
				continue
			default:
				values[k] = v
			}
		}
	}
	if len(values) == 0 {
		return fmt.Errorf("dlq entry has no replayable payload")
	}

	pipe := rdb.Pipeline()
	pipe.XAdd(ctx, &redis.XAddArgs{Stream: targetStream, Values: values})
	pipe.XDel(ctx, dlqStream, dlqMsgID)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("dlq retry pipeline: %w", err)
	}
	return nil
}

func adminapiParseDLQEntryIDFromRoute(dlqID string) string {
	const prefix = "shard-"
	if !strings.HasPrefix(dlqID, prefix) {
		return ""
	}
	rest := dlqID[len(prefix):]
	dash := strings.Index(rest, "-")
	if dash < 0 || dash+1 >= len(rest) {
		return ""
	}
	return rest[dash+1:]
}
