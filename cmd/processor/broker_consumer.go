// BrokerConsumerGroup: parallel broker partition workers -> ClickHouse StoreBatch.
// Used when CH_INGEST_SOURCE=broker; offsets persisted locally + broker CommitOffset RPC.
package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream"
	"ad-event-processor/pkg/broker"
	"ad-event-processor/pkg/broker/client"
	"ad-event-processor/pkg/logger"
)

// BrokerConsumerGroupConfig drives one consumer group across broker partitions.
type BrokerConsumerGroupConfig struct {
	BrokerAddr         string
	RedisURL           string
	Topic              string
	Group              string
	PartitionCount     int
	BatchSize          int           // events per StoreBatch flush (default 50000)
	FlushInterval      time.Duration // max wait between flushes (default 1000ms)
	MaxBytes           uint32        // max Fetch payload bytes (default 64 MiB)
	Timeout            time.Duration // broker RPC timeout (default 10s)
	DataDir            string        // local offset tracker directory
	ShadowMode         bool          // count only; skip StoreBatch (BROKER_SHADOW_MODE)
	OnMessageProcessed func(evt *domain.Event, brokerOffset uint64)
}

// BrokerConsumerGroup coordinates per-partition brokerWorker goroutines.
type BrokerConsumerGroup struct {
	cfg           BrokerConsumerGroupConfig
	store         domain.EventStore
	offsetTracker *broker.ConsumerOffsetTracker
	logger        *logger.Logger
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	workers       []*brokerWorker
}

type brokerWorker struct {
	parent    *BrokerConsumerGroup
	partition uint16
	cli       *client.Client
}

func NewBrokerConsumerGroup(
	store domain.EventStore,
	cfg BrokerConsumerGroupConfig,
	lg *logger.Logger,
) (*BrokerConsumerGroup, error) {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 50000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 1000 * time.Millisecond
	}
	if cfg.MaxBytes == 0 {
		cfg.MaxBytes = 64 * 1024 * 1024 // 64 MiB fetch cap per partition poll
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.PartitionCount <= 0 {
		cfg.PartitionCount = 1
	}
	if cfg.Topic == "" {
		cfg.Topic = "ad-events"
	}
	if cfg.Group == "" {
		cfg.Group = "clickhouse_processor"
	}

	offsetTracker, err := broker.NewConsumerOffsetTracker(cfg.DataDir)
	if err != nil {
		slog.Warn("broker consumer group offset tracker disk init warning", "dir", cfg.DataDir, "error", err)
	}

	bcg := &BrokerConsumerGroup{
		cfg:           cfg,
		store:         store,
		offsetTracker: offsetTracker,
		logger:        lg,
	}

	return bcg, nil
}

func (bc *BrokerConsumerGroup) Start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	bc.cancel = cancel

	// One brokerWorker goroutine per partition; shared offset tracker under cfg.DataDir.
	for p := range bc.cfg.PartitionCount {
		w := &brokerWorker{
			parent:    bc,
			partition: uint16(p),
			cli:       client.NewClient(bc.cfg.BrokerAddr, bc.cfg.Timeout),
		}
		if bc.cfg.RedisURL != "" {
			w.cli.SetRedisURL(bc.cfg.RedisURL)
		}
		bc.workers = append(bc.workers, w)
		bc.wg.Add(1)
		go w.run(runCtx)
	}
	slog.Info("broker consumer group started",
		"topic", bc.cfg.Topic,
		"group", bc.cfg.Group,
		"partitions", bc.cfg.PartitionCount,
		"batch_size", bc.cfg.BatchSize,
	)
}

func (w *brokerWorker) run(ctx context.Context) {
	defer w.parent.wg.Done()

	if err := w.cli.Connect(); err != nil {
		slog.Error("broker worker failed to connect", "partition", w.partition, "error", err)
		return
	}
	defer func() { _ = w.cli.Close() }()

	// Offset resume: local disk tracker first, then broker CommittedOffset RPC when disk empty.
	var start uint64
	if w.parent.offsetTracker != nil {
		if stored, err := w.parent.offsetTracker.GetCommittedOffset(w.parent.cfg.Topic, w.partition, w.parent.cfg.Group); err == nil && stored > 0 {
			start = stored
		}
	}
	if start == 0 {
		if remote, err := w.cli.CommittedOffset(w.parent.cfg.Topic, w.partition, w.parent.cfg.Group); err == nil {
			start = remote
		}
	}

	batch := make([]*domain.Event, 0, w.parent.cfg.BatchSize)
	lastFlush := time.Now()

	for {
		if ctx.Err() != nil {
			// Shutdown: flush partial batch before returning (drain uses len(batch) as commit hint).
			w.drain(ctx, start, batch)
			return
		}

		iter, err := w.cli.Fetch(ctx, w.parent.cfg.Topic, w.partition, start, w.parent.cfg.MaxBytes)
		if err != nil {
			slog.Error("broker worker fetch failed", "partition", w.partition, "error", err)
			select {
			case <-ctx.Done():
				w.drain(ctx, start, batch)
				return
			case <-time.After(time.Second):
			}
			continue
		}

		var nextCommit uint64
		processed := 0
		for iter.Next() {
			if ctx.Err() != nil {
				break
			}
			parseErr := stream.ParseBrokerPayloadStream(iter.Payload, func(evt *domain.Event) {
				metrics.BrokerIngestMessagesTotal.WithLabelValues(w.parent.cfg.Topic, w.parent.cfg.Group, evt.Type).Inc()
				if w.parent.cfg.OnMessageProcessed != nil {
					// CH broker path: fraud microbatch hook runs before StoreBatch (same as Redis _ch consumer).
					w.parent.cfg.OnMessageProcessed(evt, iter.Offset)
				}
				batch = append(batch, evt)
				processed++
			})
			if parseErr != nil {
				metrics.BrokerIngestParseErrorsTotal.WithLabelValues(w.parent.cfg.Topic, w.parent.cfg.Group).Inc()
			}
			nextCommit = iter.Offset + 1

			if len(batch) >= w.parent.cfg.BatchSize || time.Since(lastFlush) >= w.parent.cfg.FlushInterval {
				committed, flushErr := w.flushBatch(ctx, batch, nextCommit)
				if flushErr != nil {
					return
				}
				start = committed
				batch = batch[:0]
				lastFlush = time.Now()
			}
		}

		if len(batch) > 0 {
			committed, flushErr := w.flushBatch(ctx, batch, nextCommit)
			if flushErr != nil {
				return
			}
			start = committed
			batch = batch[:0]
			lastFlush = time.Now()
		} else if nextCommit > start {
			w.commitOffset(ctx, nextCommit)
			start = nextCommit
		}

		if iter.HighWatermark > start {
			metrics.BrokerConsumerLagMessages.WithLabelValues(w.parent.cfg.Topic, w.parent.cfg.Group).Set(float64(iter.HighWatermark - start))
		}

		if processed == 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}
}

func (w *brokerWorker) flushBatch(ctx context.Context, batch []*domain.Event, nextCommit uint64) (uint64, error) {
	if len(batch) == 0 {
		return nextCommit, nil
	}

	if w.parent.cfg.ShadowMode {
		// BROKER_SHADOW_MODE=1: parse and count only; skip StoreBatch until cutover.
		metrics.BrokerShadowMessagesTotal.WithLabelValues(w.parent.cfg.Topic, w.parent.cfg.Group).Add(float64(len(batch)))
		for _, evt := range batch {
			domain.EventPool.Put(evt)
		}
		return nextCommit, nil
	}

	storeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	err := w.parent.store.StoreBatch(storeCtx, batch)
	cancel()

	if err != nil {
		slog.Error("clickhouse store batch failed from broker", "partition", w.partition, "events", len(batch), "error", err)
		return 0, err // worker exits; Close/Wait on main shutdown retries via next process start offset
	}

	for _, evt := range batch {
		domain.EventPool.Put(evt)
	}

	w.commitOffset(ctx, nextCommit)
	return nextCommit, nil
}

func (w *brokerWorker) commitOffset(ctx context.Context, offset uint64) {
	// Dual commit: local mmap tracker for fast resume + broker RPC for HA consumer groups.
	if w.parent.offsetTracker != nil {
		_ = w.parent.offsetTracker.CommitOffset(w.parent.cfg.Topic, w.partition, w.parent.cfg.Group, offset)
	}
	_, _ = w.cli.CommitOffset(w.parent.cfg.Topic, w.partition, w.parent.cfg.Group, offset)
	metrics.BrokerIngestCommitsTotal.WithLabelValues(w.parent.cfg.Topic, w.parent.cfg.Group).Inc()
}

func (w *brokerWorker) drain(ctx context.Context, start uint64, batch []*domain.Event) {
	if len(batch) == 0 {
		return
	}
	_, _ = w.flushBatch(ctx, batch, start+uint64(len(batch)))
}

func (bc *BrokerConsumerGroup) Close() {
	// Cancels partition workers; Wait joins goroutines before process exit.
	if bc.cancel != nil {
		bc.cancel()
	}
}

func (bc *BrokerConsumerGroup) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		bc.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (bc *BrokerConsumerGroup) OffsetTracker() *broker.ConsumerOffsetTracker {
	return bc.offsetTracker
}
