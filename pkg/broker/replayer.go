package broker

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion"
	blog "ad-event-processor/pkg/broker/log"
)

type ReplayConfig struct {
	DataDir   string
	Topic     string
	From      time.Time
	To        time.Time
	Target    string
	CHDSN     string
	BatchSize int
}

type ReplayResult struct {
	EventsRead     int64
	EventsReplayed int64
	PayloadHash    string
}

type Replayer struct {
	cfg   ReplayConfig
	store domain.EventStore
}

func NewReplayer(cfg ReplayConfig, store domain.EventStore) *Replayer {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50000
	}
	return &Replayer{
		cfg:   cfg,
		store: store,
	}
}

func (r *Replayer) Replay(ctx context.Context) (*ReplayResult, error) {
	partDirs, err := findPartitionDirs(r.cfg.DataDir, r.cfg.Topic)
	if err != nil {
		return nil, fmt.Errorf("find partition dirs: %w", err)
	}
	if len(partDirs) == 0 {
		return nil, fmt.Errorf("no partition directories found in %s for topic %s", r.cfg.DataDir, r.cfg.Topic)
	}

	res := &ReplayResult{}
	hasher := sha256.New()

	for _, partDir := range partDirs {
		partLog, err := blog.NewPartitionLog(ctx, partDir, 1024*1024*1024, 4096)
		if err != nil {
			slog.Warn("failed to open partition log", "dir", partDir, "error", err)
			continue
		}

		batch := make([]*domain.Event, 0, r.cfg.BatchSize)
		var offset uint64 = 0

		for {
			if ctx.Err() != nil {
				_ = partLog.Close()
				return nil, ctx.Err()
			}

			payload, bufPtr, readErr := partLog.ReadRawMessages(offset, 1024*1024)
			if readErr != nil {
				if errors.Is(readErr, io.EOF) {
					break
				}
				slog.Warn("error reading raw messages from partition", "dir", partDir, "offset", offset, "error", readErr)
				break
			}

			rawOffset := 0
			for rawOffset+12 <= len(payload) {
				length := binary.BigEndian.Uint32(payload[rawOffset : rawOffset+4])
				recOffset := binary.BigEndian.Uint64(payload[rawOffset+4 : rawOffset+12])
				userPayloadLen := int(length) - 8
				if userPayloadLen < 0 || rawOffset+12+userPayloadLen > len(payload) {
					break
				}
				userPayload := payload[rawOffset+12 : rawOffset+12+userPayloadLen]

				offset = recOffset + 1

				parseErr := ingestion.ParseBrokerPayloadStream(userPayload, func(evt *domain.Event) {
					res.EventsRead++
					if !r.cfg.From.IsZero() && evt.CreatedAt.Before(r.cfg.From) {
						domain.EventPool.Put(evt)
						return
					}
					if !r.cfg.To.IsZero() && evt.CreatedAt.After(r.cfg.To) {
						domain.EventPool.Put(evt)
						return
					}

					hasher.Write([]byte(evt.ClickID))
					hasher.Write([]byte(evt.Type))

					res.EventsReplayed++
					batch = append(batch, evt)

					if len(batch) >= r.cfg.BatchSize {
						if r.store != nil {
							if err := r.store.StoreBatch(ctx, batch); err != nil {
								slog.Error("replay store batch failed", "events", len(batch), "error", err)
							}
						}
						for _, e := range batch {
							domain.EventPool.Put(e)
						}
						batch = batch[:0]
					}
				})

				if parseErr != nil {
					slog.Warn("replay payload stream parse warning", "error", parseErr)
				}

				rawOffset += 12 + userPayloadLen
			}

			if bufPtr != nil {
				blog.FetchBufPool.Put(bufPtr)
			}
		}

		if len(batch) > 0 {
			if r.store != nil {
				if err := r.store.StoreBatch(ctx, batch); err != nil {
					slog.Error("replay store final batch failed", "events", len(batch), "error", err)
				}
			}
			for _, e := range batch {
				domain.EventPool.Put(e)
			}
			batch = batch[:0]
		}

		_ = partLog.Close()
	}

	res.PayloadHash = hex.EncodeToString(hasher.Sum(nil))
	return res, nil
}

func findPartitionDirs(dataDir string, topic string) ([]string, error) {
	var dirs []string
	seen := make(map[string]bool)

	if topic != "" {
		topicDir := filepath.Join(dataDir, topic)
		entries, err := os.ReadDir(topicDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() && strings.HasPrefix(e.Name(), "partition_") {
					p := filepath.Join(topicDir, e.Name())
					if !seen[p] {
						seen[p] = true
						dirs = append(dirs, p)
					}
				}
			}
		}
	}

	entries, err := os.ReadDir(dataDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), "partition_") {
				p := filepath.Join(dataDir, e.Name())
				if !seen[p] {
					seen[p] = true
					dirs = append(dirs, p)
				}
			}
		}
	}

	sort.Strings(dirs)
	return dirs, nil
}
