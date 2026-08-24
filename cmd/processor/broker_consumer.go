package main

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/broker"
	"ad-event-processor/pkg/broker/client"
	"ad-event-processor/pkg/logger"
)

type BrokerConsumerGroupConfig struct {
	BrokerAddr         string
	RedisURL           string
	Topic              string
	Group              string
	PartitionCount     int
	BatchSize          int
	FlushInterval      time.Duration
	MaxBytes           uint32
	Timeout            time.Duration
	DataDir            string
	ShadowMode         bool
	OnMessageProcessed func(evt *domain.Event, brokerOffset uint64)
}

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
		cfg.MaxBytes = 64 * 1024 * 1024
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

func (bcg *BrokerConsumerGroup) Start(ctx context.Context) {
	runCtx, cancel := context.WithCancel(ctx)
	bcg.cancel = cancel

	for p := 0; p < bcg.cfg.PartitionCount; p++ {
		w := &brokerWorker{
			parent:    bcg,
			partition: uint16(p),
			cli:       client.NewClient(bcg.cfg.BrokerAddr, bcg.cfg.Timeout),
		}
		if bcg.cfg.RedisURL != "" {
			w.cli.SetRedisURL(bcg.cfg.RedisURL)
		}
		bcg.workers = append(bcg.workers, w)
		bcg.wg.Add(1)
		go w.run(runCtx)
	}
	slog.Info("broker consumer group started",
		"topic", bcg.cfg.Topic,
		"group", bcg.cfg.Group,
		"partitions", bcg.cfg.PartitionCount,
		"batch_size", bcg.cfg.BatchSize,
	)
}

func (w *brokerWorker) run(ctx context.Context) {
	defer w.parent.wg.Done()

	if err := w.cli.Connect(); err != nil {
		slog.Error("broker worker failed to connect", "partition", w.partition, "error", err)
		return
	}
	defer func() { _ = w.cli.Close() }()

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
			parseErr := ingestion.ParseBrokerPayloadStream(iter.Payload, func(evt *domain.Event) {
				metrics.BrokerIngestMessagesTotal.WithLabelValues(w.parent.cfg.Topic, w.parent.cfg.Group, evt.Type).Inc()
				if w.parent.cfg.OnMessageProcessed != nil {
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
		return 0, err
	}

	for _, evt := range batch {
		domain.EventPool.Put(evt)
	}

	w.commitOffset(ctx, nextCommit)
	return nextCommit, nil
}

func (w *brokerWorker) commitOffset(ctx context.Context, offset uint64) {
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

func (bcg *BrokerConsumerGroup) Close() {
	if bcg.cancel != nil {
		bcg.cancel()
	}
}

func (bcg *BrokerConsumerGroup) Wait(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		bcg.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (bcg *BrokerConsumerGroup) OffsetTracker() *broker.ConsumerOffsetTracker {
	return bcg.offsetTracker
}
