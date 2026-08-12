package ingestion

import (
	"context"
	"errors"
	"fmt"
	"hash/crc32"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/pkg/logger"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

const defaultSettlementLaneBuffer = 4096

type laneMsg struct {
	event *domain.Event
	msgID string
}

type SettlementWorker struct {
	consumer   *StreamConsumer
	lanes      int
	laneCh     []chan laneMsg
	laneWG     sync.WaitGroup
	startMu    sync.Mutex
	started    bool
	flushInt   time.Duration
	batchSize  int
	consumerID string
}

func NewSettlementWorker(
	store domain.EventStore,
	rdb redis.UniversalClient,
	streamName, groupName, consumerID string,
	lanes, batchSize int,
	flushInt, writeTimeout time.Duration,
	retryInitWait, retryMaxWait time.Duration,
	maxRetries int,
	streamMinIdle time.Duration,
	drainTimeout time.Duration,
) *SettlementWorker {
	if lanes <= 0 {
		lanes = 1
	}
	if batchSize <= 0 {
		batchSize = 1000
	}
	if flushInt <= 0 {
		flushInt = 100 * time.Millisecond
	}
	base := NewStreamConsumer(
		store, rdb, streamName, groupName, consumerID,
		batchSize, 0,
		flushInt, writeTimeout,
		retryInitWait, retryMaxWait, maxRetries,
		streamMinIdle, drainTimeout,
	)
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	uniqueID := fmt.Sprintf("%s-settle-%s", consumerID, hostname)
	laneCh := make([]chan laneMsg, lanes)
	for i := range laneCh {
		laneCh[i] = make(chan laneMsg, defaultSettlementLaneBuffer)
	}
	return &SettlementWorker{
		consumer:   base,
		lanes:      lanes,
		laneCh:     laneCh,
		flushInt:   flushInt,
		batchSize:  batchSize,
		consumerID: uniqueID,
	}
}

func (w *SettlementWorker) SetLogger(l *logger.Logger) {
	if w != nil && w.consumer != nil {
		w.consumer.SetLogger(l)
	}
}

func (w *SettlementWorker) SetAuditLogSampleMask(mask int) {
	if w != nil && w.consumer != nil {
		w.consumer.SetAuditLogSampleMask(mask)
	}
}

func (w *SettlementWorker) SetWeightController(ctrl *ProcessorWeightController) {
	if w != nil && w.consumer != nil {
		w.consumer.SetWeightController(ctrl)
	}
}

func (w *SettlementWorker) SetOnMessageProcessed(cb func(evt *domain.Event, msgID string)) {
	if w != nil && w.consumer != nil {
		w.consumer.SetOnMessageProcessed(cb)
	}
}

func settlementLaneIndex(campaignID uuid.UUID, lanes int) int {
	if lanes <= 1 {
		return 0
	}
	return int(crc32.ChecksumIEEE(campaignID[:]) % uint32(lanes))
}

func (w *SettlementWorker) Start(ctx context.Context) {
	if w == nil || w.consumer == nil {
		return
	}
	w.startMu.Lock()
	defer w.startMu.Unlock()
	if w.started {
		return
	}
	w.started = true

	procCtx, cancel := context.WithCancel(ctx)
	w.consumer.cancel = cancel

	err := w.consumer.rdb.XGroupCreateMkStream(procCtx, w.consumer.streamName, w.consumer.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		slog.Error("failed to create settlement consumer group", "error", err, "stream", w.consumer.streamName, "group", w.consumer.groupName)
	}

	for i := 0; i < w.lanes; i++ {
		w.laneWG.Add(1)
		go func(laneIdx int) {
			defer w.laneWG.Done()
			w.runLane(procCtx, laneIdx)
		}(i)
	}

	w.consumer.wg.Add(1)
	go func() {
		defer w.consumer.wg.Done()
		w.readLoop(procCtx)
	}()

	w.consumer.StartMaintenance(procCtx)
}

func (w *SettlementWorker) Close() {
	if w != nil && w.consumer != nil && w.consumer.cancel != nil {
		w.consumer.cancel()
	}
}

func (w *SettlementWorker) Wait(ctx context.Context) error {
	if w == nil || w.consumer == nil {
		return nil
	}
	done := make(chan struct{})
	go func() {
		w.laneWG.Wait()
		w.consumer.wg.Wait()
		close(done)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		return nil
	}
}

func (w *SettlementWorker) readLoop(ctx context.Context) {
	workerID := w.consumerID + "-reader"
	initCtx, initCancel := context.WithTimeout(context.Background(), w.consumer.writeTimeout*2)
	w.consumer.recoverPending(initCtx, workerID)
	initCancel()

	xreadArgs := &redis.XReadGroupArgs{
		Group:    w.consumer.groupName,
		Consumer: workerID,
		Streams:  []string{w.consumer.streamName, ">"},
		Count:    int64(w.batchSize),
		Block:    200 * time.Millisecond,
	}

	for {
		if w.consumer.pauseStreamReads(ctx) {
			continue
		}
		select {
		case <-ctx.Done():
			return
		default:
		}

		if w.consumer.weightCtrl != nil {
			w.consumer.weightCtrl.ThrottleBeforeRead(ctx)
		}

		streams, err := w.consumer.rdb.XReadGroup(ctx, xreadArgs).Result()
		if err != nil {
			if err == redis.Nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				continue
			}
			slog.Error("settlement read loop failed", "error", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				evt := w.consumer.ParseMessage(msg.ID, msg.Values)
				lane := settlementLaneIndex(evt.CampaignID, w.lanes)
				if w.consumer.onMessageProcessed != nil {
					w.consumer.onMessageProcessed(evt, msg.ID)
				}
				select {
				case <-ctx.Done():
					return
				case w.laneCh[lane] <- laneMsg{event: evt, msgID: msg.ID}:
					metrics.SettlementLaneDepth.WithLabelValues(strconv.Itoa(lane)).Set(float64(len(w.laneCh[lane])))
				}
			}
		}
	}
}

func (w *SettlementWorker) runLane(ctx context.Context, laneIdx int) {
	workerID := fmt.Sprintf("%s-lane-%d", w.consumerID, laneIdx)
	laneLabel := strconv.Itoa(laneIdx)
	batch := make([]*domain.Event, 0, w.batchSize)
	msgIDs := make([]string, 0, w.batchSize)
	lastFlush := time.Now()
	ticker := time.NewTicker(w.flushInt)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if len(batch) > 0 && !batch[0].CreatedAt.IsZero() {
			metrics.SettlementLagSeconds.Set(time.Since(batch[0].CreatedAt).Seconds())
		}
		if err := w.consumer.FlushBatch(ctx, batch, msgIDs, workerID); err != nil {
			if !errors.Is(err, context.Canceled) {
				w.consumer.recordFailure(workerID)
				slog.Error("settlement lane flush failed", "lane", laneIdx, "error", err, "batch_size", len(batch))
			}
			for _, e := range batch {
				domain.EventPool.Put(e)
			}
		} else {
			w.consumer.recordSuccess(workerID)
			for _, e := range batch {
				domain.EventPool.Put(e)
			}
		}
		batch = batch[:0]
		msgIDs = msgIDs[:0]
		lastFlush = time.Now()
		metrics.SettlementLaneDepth.WithLabelValues(laneLabel).Set(float64(len(w.laneCh[laneIdx])))
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case <-ticker.C:
			if len(batch) > 0 && time.Since(lastFlush) >= w.flushInt {
				flush()
			}
		case item, ok := <-w.laneCh[laneIdx]:
			if !ok {
				flush()
				return
			}
			batch = append(batch, item.event)
			msgIDs = append(msgIDs, item.msgID)
			metrics.SettlementLaneDepth.WithLabelValues(laneLabel).Set(float64(len(w.laneCh[laneIdx])))
			if len(batch) >= w.batchSize {
				flush()
			}
		}
	}
}
