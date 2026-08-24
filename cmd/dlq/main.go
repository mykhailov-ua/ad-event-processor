package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"

	"ad-event-processor/internal/ingestion"
	"ad-event-processor/internal/ingestion/pb"
	"ad-event-processor/pkg/lifecycle"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/proto"
)

func fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}

func main() {
	var (
		action    = flag.String("action", "archive", "Action to perform: archive, requeue, restore, inspect, or edit")
		stream    = flag.String("stream", "ad:events:dlq", "DLQ stream name or target stream name")
		dest      = flag.String("dest", "dlq_archive.bin", "Destination file for archive/restore or target stream name for requeue")
		batch     = flag.Int64("batch", 1000, "Batch size for processing")
		redisURL  = flag.String("redis", "redis://localhost:6379", "Redis connection string")
		rateLimit = flag.Int64("rate", 0, "Rate limit (events per second) for requeue/restore. 0 means unlimited.")
		id        = flag.String("id", "", "ID of the stream message to edit (required for -action=edit)")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	ctx, cancel := lifecycle.NotifyContext(context.Background())
	defer cancel()

	opt, err := redis.ParseURL(*redisURL)
	if err != nil {
		fatal("invalid redis url", "error", err)
	}
	redisClient := redis.NewClient(opt)
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		fatal("failed to connect to redis", "error", err)
	}

	switch *action {
	case "archive":
		if err := archiveDLQ(ctx, redisClient, *stream, *dest, *batch); err != nil {
			fatal("archive failed", "error", err)
		}
	case "requeue":
		if err := requeueDLQ(ctx, redisClient, *stream, *dest, *batch, *rateLimit); err != nil {
			fatal("requeue failed", "error", err)
		}
	case "restore":
		if err := restoreDLQ(ctx, redisClient, *dest, *stream, *batch, *rateLimit); err != nil {
			fatal("restore failed", "error", err)
		}
	case "inspect":
		if err := inspectStream(ctx, redisClient, *stream, *batch); err != nil {
			fatal("inspect failed", "error", err)
		}
	case "edit":
		if *id == "" {
			fatal("message id required for edit action", "flag", "-id")
		}
		if err := editDLQMessage(ctx, redisClient, *stream, *id); err != nil {
			fatal("edit failed", "error", err)
		}
	default:
		fatal("unknown action", "action", *action)
	}
}

func archiveDLQ(ctx context.Context, redisClient *redis.Client, stream, destFile string, batchSize int64) error {
	file, err := os.OpenFile(destFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := bufio.NewWriter(file)
	defer func() { _ = writer.Flush() }()

	startID := "0-0"
	var totalProcessed int64

	slog.Info("starting dlq archive", "stream", stream, "dest", destFile)

	pbDLQ := &pb.AdDLQEvent{}
	pbStream := &pb.AdStreamEvent{}

	for {
		msgs, err := redisClient.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream, startID},
			Count:   batchSize,
			Block:   time.Millisecond * 10,
		}).Result()

		if err != nil && !errors.Is(err, redis.Nil) {
			return fmt.Errorf("failed to read from stream: %w", err)
		}

		if len(msgs) == 0 || len(msgs[0].Messages) == 0 {
			break
		}

		pipe := redisClient.Pipeline()
		var msgIDs []string

		for _, msg := range msgs[0].Messages {
			pbDLQ.Reset()

			if rawBytesStr, ok := msg.Values["d"].(string); ok {
				if err := proto.Unmarshal(ingestion.UnsafeBytes(rawBytesStr), pbDLQ); err != nil {
					pbStream.Reset()
					if err := proto.Unmarshal(ingestion.UnsafeBytes(rawBytesStr), pbStream); err == nil {
						pbDLQ.OriginalEvent = pbStream
						pbDLQ.Error = ingestion.UnsafeBytes("recovered stream event")
						pbDLQ.OriginalId = ingestion.UnsafeBytes(msg.ID)
						pbDLQ.FailedAtUnix = time.Now().Unix()
					} else {
						pbStream.Reset()
						pbStream.Payload = ingestion.UnsafeBytes(rawBytesStr)
						pbDLQ.OriginalEvent = pbStream
						pbDLQ.Error = ingestion.UnsafeBytes("unknown binary")
						pbDLQ.OriginalId = ingestion.UnsafeBytes(msg.ID)
						pbDLQ.FailedAtUnix = time.Now().Unix()
					}
				}
			} else {
				pbStream.Reset()
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
				switch v := msg.Values["payload"].(type) {
				case string:
					pbStream.Payload = ingestion.UnsafeBytes(v)
				case []byte:
					pbStream.Payload = v
				}
				if v, ok := msg.Values["ip"].(string); ok {
					pbStream.Ip = ingestion.UnsafeBytes(v)
				}
				if v, ok := msg.Values["ua"].(string); ok {
					pbStream.Ua = ingestion.UnsafeBytes(v)
				}
				if v, ok := msg.Values["created_at_unix"].(int64); ok {
					pbStream.CreatedAtUnix = v
				}

				pbDLQ.OriginalEvent = pbStream
				if v, ok := msg.Values["error"].(string); ok {
					pbDLQ.Error = ingestion.UnsafeBytes(v)
				}
				if v, ok := msg.Values["original_id"].(string); ok {
					pbDLQ.OriginalId = ingestion.UnsafeBytes(v)
				} else {
					pbDLQ.OriginalId = ingestion.UnsafeBytes(msg.ID)
				}
				if v, ok := msg.Values["failed_at_unix"].(int64); ok {
					pbDLQ.FailedAtUnix = v
				} else {
					pbDLQ.FailedAtUnix = time.Now().Unix()
				}
				if v, ok := msg.Values["worker_id"].(string); ok {
					pbDLQ.WorkerId = ingestion.UnsafeBytes(v)
				}
				if v, ok := msg.Values["retry_count"].(int64); ok {
					pbDLQ.RetryCount = int32(v)
				}
			}

			data, err := proto.Marshal(pbDLQ)
			if err != nil {
				return fmt.Errorf("failed to marshal message %s: %w", msg.ID, err)
			}

			var lengthBuf [4]byte
			binary.BigEndian.PutUint32(lengthBuf[:], uint32(len(data)))
			if _, err := writer.Write(lengthBuf[:]); err != nil {
				return fmt.Errorf("failed to write length prefix for msg %s: %w", msg.ID, err)
			}
			if _, err := writer.Write(data); err != nil {
				return fmt.Errorf("failed to write message data for msg %s: %w", msg.ID, err)
			}

			msgIDs = append(msgIDs, msg.ID)
			startID = msg.ID
		}

		pipe.XDel(ctx, stream, msgIDs...)
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to delete archived messages: %w", err)
		}

		totalProcessed += int64(len(msgIDs))
		slog.Info("dlq archive batch complete",
			"batch", len(msgIDs),
			"total", totalProcessed,
		)
	}

	slog.Info("dlq archive complete", "total", totalProcessed)
	return nil
}

func requeueDLQ(ctx context.Context, redisClient *redis.Client, dlqStream, targetStream string, batchSize, rateLimit int64) error {
	startID := "0-0"
	var totalProcessed int64

	slog.Info("starting dlq requeue",
		"dlq_stream", dlqStream,
		"target_stream", targetStream,
		"rate_limit", rateLimit,
	)

	pbDLQ := &pb.AdDLQEvent{}

	var limiter *rate.Limiter
	if rateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(rateLimit), int(rateLimit))
	}

	for {
		msgs, err := redisClient.XRead(ctx, &redis.XReadArgs{
			Streams: []string{dlqStream, startID},
			Count:   batchSize,
			Block:   time.Millisecond * 10,
		}).Result()

		if err != nil && !errors.Is(err, redis.Nil) {
			return fmt.Errorf("failed to read from stream: %w", err)
		}

		if len(msgs) == 0 || len(msgs[0].Messages) == 0 {
			break
		}

		pipe := redisClient.Pipeline()
		var msgIDs []string

		for _, msg := range msgs[0].Messages {
			if limiter != nil {
				if err := limiter.Wait(ctx); err != nil {
					return fmt.Errorf("rate limiter wait error: %w", err)
				}
			}

			values := make(map[string]interface{})
			if rawBytesStr, ok := msg.Values["d"].(string); ok {
				pbDLQ.Reset()
				if err := proto.Unmarshal(ingestion.UnsafeBytes(rawBytesStr), pbDLQ); err == nil && pbDLQ.OriginalEvent != nil {
					data, err := proto.Marshal(pbDLQ.OriginalEvent)
					if err == nil {
						values["d"] = ingestion.UnsafeString(data)
					} else {
						slog.Warn("failed to re-marshal original event from dlq message",
							"message_id", msg.ID,
							"error", err,
						)
					}
				} else {
					slog.Warn("failed to unmarshal protobuf dlq message",
						"message_id", msg.ID,
						"error", err,
					)
				}
			} else {
				for k, v := range msg.Values {
					if k != "error" && k != "original_id" && k != "failed_at" && k != "service" && k != "worker_id" && k != "retry_count" {
						values[k] = v
					}
				}
			}

			pipe.XAdd(ctx, &redis.XAddArgs{
				Stream: targetStream,
				Values: values,
			})
			msgIDs = append(msgIDs, msg.ID)
			startID = msg.ID
		}

		pipe.XDel(ctx, dlqStream, msgIDs...)
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to requeue messages: %w", err)
		}

		totalProcessed += int64(len(msgIDs))
		slog.Info("dlq requeue batch complete",
			"batch", len(msgIDs),
			"total", totalProcessed,
		)
	}

	slog.Info("dlq requeue complete", "total", totalProcessed)
	return nil
}

func restoreDLQ(ctx context.Context, redisClient *redis.Client, srcFile, targetStream string, batchSize, rateLimit int64) error {
	file, err := os.Open(srcFile)
	if err != nil {
		return fmt.Errorf("failed to open archive file: %w", err)
	}
	defer func() { _ = file.Close() }()

	slog.Info("starting dlq restore",
		"src", srcFile,
		"target_stream", targetStream,
		"rate_limit", rateLimit,
	)

	reader := bufio.NewReader(file)
	var totalProcessed int64
	var lengthBuf [4]byte
	pipe := redisClient.Pipeline()
	batchCount := 0

	pbDLQ := &pb.AdDLQEvent{}

	var limiter *rate.Limiter
	if rateLimit > 0 {
		limiter = rate.NewLimiter(rate.Limit(rateLimit), int(rateLimit))
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := reader.Read(lengthBuf[:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("failed to read length prefix: %w", err)
		}

		length := binary.BigEndian.Uint32(lengthBuf[:])
		data := make([]byte, length)
		if _, err := io.ReadFull(reader, data); err != nil {
			return fmt.Errorf("failed to read message payload: %w", err)
		}

		pbDLQ.Reset()
		if err := proto.Unmarshal(data, pbDLQ); err != nil {
			return fmt.Errorf("failed to unmarshal AdDLQEvent: %w", err)
		}

		if pbDLQ.OriginalEvent == nil {
			slog.Warn("dlq event has no original event, skipping", "offset", totalProcessed)
			continue
		}

		streamData, err := proto.Marshal(pbDLQ.OriginalEvent)
		if err != nil {
			return fmt.Errorf("failed to marshal original event: %w", err)
		}

		if limiter != nil {
			if err := limiter.Wait(ctx); err != nil {
				return fmt.Errorf("rate limiter wait error: %w", err)
			}
		}

		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: targetStream,
			Values: map[string]interface{}{
				"d": ingestion.UnsafeString(streamData),
			},
		})
		batchCount++
		totalProcessed++

		if int64(batchCount) >= batchSize {
			if _, err := pipe.Exec(ctx); err != nil {
				return fmt.Errorf("failed to restore batch to Redis: %w", err)
			}
			slog.Info("dlq restore batch complete",
				"batch", batchCount,
				"total", totalProcessed,
			)
			pipe = redisClient.Pipeline()
			batchCount = 0
		}
	}

	if batchCount > 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("failed to restore final batch to Redis: %w", err)
		}
		slog.Info("dlq restore batch complete",
			"batch", batchCount,
			"total", totalProcessed,
		)
	}

	slog.Info("dlq restore complete", "total", totalProcessed)
	return nil
}

func inspectStream(ctx context.Context, redisClient *redis.Client, stream string, batchSize int64) error {
	startID := "0-0"
	var totalProcessed int64

	slog.Info("starting dlq inspect", "stream", stream)

	pbDLQ := &pb.AdDLQEvent{}
	pbStream := &pb.AdStreamEvent{}

	for {
		msgs, err := redisClient.XRead(ctx, &redis.XReadArgs{
			Streams: []string{stream, startID},
			Count:   batchSize,
			Block:   time.Millisecond * 10,
		}).Result()

		if err != nil && !errors.Is(err, redis.Nil) {
			return fmt.Errorf("failed to read from stream: %w", err)
		}

		if len(msgs) == 0 || len(msgs[0].Messages) == 0 {
			break
		}

		for _, msg := range msgs[0].Messages {
			_, _ = fmt.Fprintf(os.Stdout, "\nMessage ID: %s\n", msg.ID)

			if rawBytesStr, ok := msg.Values["d"].(string); ok {
				pbDLQ.Reset()
				if err := proto.Unmarshal(ingestion.UnsafeBytes(rawBytesStr), pbDLQ); err == nil && pbDLQ.OriginalEvent != nil {
					_, _ = fmt.Fprintf(os.Stdout, "Format: Protobuf (AdDLQEvent)\n")
					orig := pbDLQ.OriginalEvent
					var campUUIDStr string
					if len(orig.CampaignId) == 16 {
						if u, err := uuid.FromBytes(orig.CampaignId); err == nil {
							campUUIDStr = u.String()
						}
					}
					if campUUIDStr == "" {
						campUUIDStr = ingestion.UnsafeString(orig.CampaignId)
					}

					m := map[string]interface{}{
						"error":          ingestion.UnsafeString(pbDLQ.Error),
						"original_id":    ingestion.UnsafeString(pbDLQ.OriginalId),
						"failed_at_unix": pbDLQ.FailedAtUnix,
						"failed_at":      time.Unix(pbDLQ.FailedAtUnix, 0).Format(time.RFC3339),
						"worker_id":      ingestion.UnsafeString(pbDLQ.WorkerId),
						"retry_count":    pbDLQ.RetryCount,
						"original_event": map[string]interface{}{
							"click_id":        ingestion.UnsafeString(orig.ClickId),
							"campaign_id":     campUUIDStr,
							"event_type":      ingestion.UnsafeString(orig.EventType),
							"payload":         ingestion.UnsafeString(orig.Payload),
							"ip":              ingestion.UnsafeString(orig.Ip),
							"ua":              ingestion.UnsafeString(orig.Ua),
							"created_at_unix": orig.CreatedAtUnix,
							"created_at":      time.Unix(orig.CreatedAtUnix, 0).Format(time.RFC3339),
						},
					}
					prettyJSON, _ := json.MarshalIndent(m, "", " ")
					_, _ = fmt.Fprintf(os.Stdout, "%s\n", string(prettyJSON))
				} else {
					pbStream.Reset()
					if err := proto.Unmarshal(ingestion.UnsafeBytes(rawBytesStr), pbStream); err == nil {
						_, _ = fmt.Fprintf(os.Stdout, "Format: Protobuf (AdStreamEvent)\n")
						var campUUIDStr string
						if len(pbStream.CampaignId) == 16 {
							if u, err := uuid.FromBytes(pbStream.CampaignId); err == nil {
								campUUIDStr = u.String()
							}
						}
						if campUUIDStr == "" {
							campUUIDStr = ingestion.UnsafeString(pbStream.CampaignId)
						}

						m := map[string]interface{}{
							"click_id":        ingestion.UnsafeString(pbStream.ClickId),
							"campaign_id":     campUUIDStr,
							"event_type":      ingestion.UnsafeString(pbStream.EventType),
							"payload":         ingestion.UnsafeString(pbStream.Payload),
							"ip":              ingestion.UnsafeString(pbStream.Ip),
							"ua":              ingestion.UnsafeString(pbStream.Ua),
							"created_at_unix": pbStream.CreatedAtUnix,
							"created_at":      time.Unix(pbStream.CreatedAtUnix, 0).Format(time.RFC3339),
						}
						prettyJSON, _ := json.MarshalIndent(m, "", " ")
						_, _ = fmt.Fprintf(os.Stdout, "%s\n", string(prettyJSON))
					} else {
						_, _ = fmt.Fprintf(os.Stdout, "Format: Unknown Binary Protobuf\n")
						_, _ = fmt.Fprintf(os.Stdout, "Raw values: %+v\n", msg.Values)
					}
				}
			} else {
				_, _ = fmt.Fprintf(os.Stdout, "Format: Legacy Flat Map\n")
				prettyJSON, _ := json.MarshalIndent(msg.Values, "", " ")
				_, _ = fmt.Fprintf(os.Stdout, "%s\n", string(prettyJSON))
			}
			startID = msg.ID
			totalProcessed++
		}
	}
	slog.Info("dlq inspect complete", "total", totalProcessed)
	return nil
}

type EditableStreamEvent struct {
	ClickID       string `json:"click_id"`
	CampaignID    string `json:"campaign_id"`
	EventType     string `json:"event_type"`
	Payload       string `json:"payload"`
	IP            string `json:"ip"`
	Ua            string `json:"ua"`
	CreatedAtUnix int64  `json:"created_at_unix"`
}

type EditableDLQEvent struct {
	ID            string              `json:"id"`
	Error         string              `json:"error"`
	OriginalID    string              `json:"original_id"`
	FailedAtUnix  int64               `json:"failed_at_unix"`
	WorkerID      string              `json:"worker_id"`
	RetryCount    int32               `json:"retry_count"`
	OriginalEvent EditableStreamEvent `json:"original_event"`
}

func toEditable(id string, pbDLQ *pb.AdDLQEvent) EditableDLQEvent {
	var orig EditableStreamEvent
	if pbDLQ.OriginalEvent != nil {
		campUUIDStr := ""
		if len(pbDLQ.OriginalEvent.CampaignId) == 16 {
			if u, err := uuid.FromBytes(pbDLQ.OriginalEvent.CampaignId); err == nil {
				campUUIDStr = u.String()
			}
		}
		if campUUIDStr == "" {
			campUUIDStr = ingestion.UnsafeString(pbDLQ.OriginalEvent.CampaignId)
		}

		orig = EditableStreamEvent{
			ClickID:       ingestion.UnsafeString(pbDLQ.OriginalEvent.ClickId),
			CampaignID:    campUUIDStr,
			EventType:     ingestion.UnsafeString(pbDLQ.OriginalEvent.EventType),
			Payload:       ingestion.UnsafeString(pbDLQ.OriginalEvent.Payload),
			IP:            ingestion.UnsafeString(pbDLQ.OriginalEvent.Ip),
			Ua:            ingestion.UnsafeString(pbDLQ.OriginalEvent.Ua),
			CreatedAtUnix: pbDLQ.OriginalEvent.CreatedAtUnix,
		}
	}
	return EditableDLQEvent{
		ID:            id,
		Error:         ingestion.UnsafeString(pbDLQ.Error),
		OriginalID:    ingestion.UnsafeString(pbDLQ.OriginalId),
		FailedAtUnix:  pbDLQ.FailedAtUnix,
		WorkerID:      ingestion.UnsafeString(pbDLQ.WorkerId),
		RetryCount:    pbDLQ.RetryCount,
		OriginalEvent: orig,
	}
}

func fromEditable(edit EditableDLQEvent) *pb.AdDLQEvent {
	var campID []byte
	if u, err := uuid.Parse(edit.OriginalEvent.CampaignID); err == nil {
		campID = u[:]
	} else {
		campID = ingestion.UnsafeBytes(edit.OriginalEvent.CampaignID)
	}

	return &pb.AdDLQEvent{
		Error:        ingestion.UnsafeBytes(edit.Error),
		OriginalId:   ingestion.UnsafeBytes(edit.OriginalID),
		FailedAtUnix: edit.FailedAtUnix,
		WorkerId:     ingestion.UnsafeBytes(edit.WorkerID),
		RetryCount:   edit.RetryCount,
		OriginalEvent: &pb.AdStreamEvent{
			ClickId:       ingestion.UnsafeBytes(edit.OriginalEvent.ClickID),
			CampaignId:    campID,
			EventType:     ingestion.UnsafeBytes(edit.OriginalEvent.EventType),
			Payload:       ingestion.UnsafeBytes(edit.OriginalEvent.Payload),
			Ip:            ingestion.UnsafeBytes(edit.OriginalEvent.IP),
			Ua:            ingestion.UnsafeBytes(edit.OriginalEvent.Ua),
			CreatedAtUnix: edit.OriginalEvent.CreatedAtUnix,
		},
	}
}

func launchEditor(filepath string) error {
	editor := os.Getenv("EDITOR")
	if editor != "" {
		cmd := exec.Command(editor, filepath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	for _, ed := range []string{"nano", "vim", "vi"} {
		cmd := exec.Command(ed, filepath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to start editor: please set your EDITOR environment variable")
}

func editDLQMessage(ctx context.Context, redisClient *redis.Client, stream, id string) error {
	msgs, err := redisClient.XRange(ctx, stream, id, id).Result()
	if err != nil {
		return fmt.Errorf("failed to fetch message %s from stream: %w", id, err)
	}
	if len(msgs) == 0 {
		return fmt.Errorf("message %s not found in stream %s", id, stream)
	}
	msg := msgs[0]

	rawBytesStr, ok := msg.Values["d"].(string)
	if !ok {
		return fmt.Errorf("message %s does not contain data field 'd'", id)
	}

	pbDLQ := &pb.AdDLQEvent{}
	if err := proto.Unmarshal(ingestion.UnsafeBytes(rawBytesStr), pbDLQ); err != nil {
		return fmt.Errorf("failed to unmarshal AdDLQEvent: %w", err)
	}

	editable := toEditable(id, pbDLQ)
	jsonData, err := json.MarshalIndent(editable, "", " ")
	if err != nil {
		return fmt.Errorf("failed to marshal editable event to JSON: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "dlq-edit-*.json")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(jsonData); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write JSON to temporary file: %w", err)
	}
	_ = tmpFile.Close()

	slog.Info("opening editor for dlq message", "message_id", id)
	if err := launchEditor(tmpPath); err != nil {
		return err
	}

	modifiedData, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to read modified file: %w", err)
	}

	var modifiedEditable EditableDLQEvent
	if err := json.Unmarshal(modifiedData, &modifiedEditable); err != nil {
		return fmt.Errorf("failed to parse modified JSON: %w", err)
	}

	modifiedPBDLQ := fromEditable(modifiedEditable)
	newRawBytes, err := proto.Marshal(modifiedPBDLQ)
	if err != nil {
		return fmt.Errorf("failed to marshal modified event to Protobuf: %w", err)
	}

	pipe := redisClient.Pipeline()
	pipe.XDel(ctx, stream, id)
	pipe.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"d": ingestion.UnsafeString(newRawBytes),
		},
	})
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to update message in stream: %w", err)
	}

	newID := cmds[1].(*redis.StringCmd).Val()
	slog.Info("dlq message updated", "old_id", id, "new_id", newID)
	return nil
}
