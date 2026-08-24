package controlplane

import (
	"context"
	"sync"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingestion/pb"
	"ad-event-processor/pkg/broker/client"

	"github.com/google/uuid"
)

var budgetDeltaPool = sync.Pool{
	New: func() any { return &pb.BudgetDelta{} },
}

type BudgetDeltaConsumer struct {
	aggregator *domain.BudgetDeltaAggregator
	cfg        domain.BrokerConsumerConfig
	cli        *client.Client
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewBudgetDeltaConsumer(agg *domain.BudgetDeltaAggregator, cfg domain.BrokerConsumerConfig) *BudgetDeltaConsumer {
	if agg == nil || cfg.BrokerAddr == "" || cfg.Topic == "" {
		return nil
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.IdleWait <= 0 {
		cfg.IdleWait = 250 * time.Millisecond
	}
	return &BudgetDeltaConsumer{
		aggregator: agg,
		cfg:        cfg,
		cli:        client.NewClient(cfg.BrokerAddr, cfg.Timeout),
	}
}

func (c *BudgetDeltaConsumer) Start(ctx context.Context) {
	if c == nil {
		return
	}
	if c.cfg.RedisURL != "" {
		c.cli.SetRedisURL(c.cfg.RedisURL)
	}
	runCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.run(runCtx)
	}()
}

func (c *BudgetDeltaConsumer) Close() {
	if c == nil {
		return
	}
	if c.cancel != nil {
		c.cancel()
	}
	c.wg.Wait()
	_ = c.cli.Close()
}

func (c *BudgetDeltaConsumer) run(ctx context.Context) {
	if err := c.cli.Connect(); err != nil {
		return
	}
	offset, _ := c.cli.CommittedOffset(c.cfg.Topic, c.cfg.Partition, c.cfg.Group)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		iter, err := c.cli.Fetch(ctx, c.cfg.Topic, c.cfg.Partition, offset, c.cfg.MaxBytes)
		if err != nil {
			time.Sleep(c.cfg.IdleWait)
			continue
		}
		processed := 0
		for iter.Next() {
			c.ingest(iter.Payload)
			offset = iter.Offset + 1
			processed++
		}
		if processed == 0 {
			time.Sleep(c.cfg.IdleWait)
			continue
		}
		_, _ = c.cli.CommitOffset(c.cfg.Topic, c.cfg.Partition, c.cfg.Group, offset)
	}
}

func (c *BudgetDeltaConsumer) ingest(payload []byte) {
	msg := budgetDeltaPool.Get().(*pb.BudgetDelta)
	msg.Reset()
	if err := msg.UnmarshalVT(payload); err != nil {
		budgetDeltaPool.Put(msg)
		return
	}
	if len(msg.CampaignId) >= 16 {
		var id uuid.UUID
		copy(id[:], msg.CampaignId[:16])
		c.aggregator.Record(id, msg.AmountMicro)
	}
	budgetDeltaPool.Put(msg)
}
