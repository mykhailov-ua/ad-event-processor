package ingest

import (
	"context"
	"sync/atomic"

	"ad-event-processor/internal/domain"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	redis "github.com/redis/go-redis/v9"
)

type mockBatch struct {
	driver.Batch
	appendFn func(args ...any) error
	sendFn   func() error
}

func (m *mockBatch) Append(v ...any) error {
	if m.appendFn != nil {
		return m.appendFn(v...)
	}
	return nil
}

func (m *mockBatch) Send() error {
	if m.sendFn != nil {
		return m.sendFn()
	}
	return nil
}

type mockConn struct {
	driver.Conn
	prepareBatchFn func(ctx context.Context, query string) (driver.Batch, error)
	closeFn        func() error
}

func (m *mockConn) PrepareBatch(ctx context.Context, query string, opts ...driver.PrepareBatchOption) (driver.Batch, error) {
	if m.prepareBatchFn != nil {
		return m.prepareBatchFn(ctx, query)
	}
	return nil, nil
}

func (m *mockConn) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func benchRegistryForCampaign(camp *domain.Campaign) *Registry {
	reg := NewRegistry(nil)
	enrichMockCampaign(camp)
	reg.SeedCampaignForTest(camp)
	return reg
}

type countingRedisXAdd struct {
	mockRedisClient
	xadds atomic.Int32
}

func (m *countingRedisXAdd) Pipeline() redis.Pipeliner {
	parent := m
	return &countingPipeliner{
		mockPipeliner: mockPipeliner{
			incrCmd: redis.NewIntCmd(context.Background()),
			doCmd:   redis.NewCmd(context.Background()),
		},
		parent: parent,
	}
}

type countingPipeliner struct {
	mockPipeliner
	parent *countingRedisXAdd
}

func (p *countingPipeliner) XAdd(ctx context.Context, args *redis.XAddArgs) *redis.StringCmd {
	p.parent.xadds.Add(1)
	return p.mockPipeliner.XAdd(ctx, args)
}
