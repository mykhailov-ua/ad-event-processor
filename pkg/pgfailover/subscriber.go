package pgfailover

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type ReconnectFunc func(pool *pgxpool.Pool)

type Subscriber struct {
	rdb          redis.UniversalClient
	fencing      *FencingGate
	reconnect    ReconnectFunc
	maxConns     int
	minConns     int
	interval     time.Duration
	closeCh      chan struct{}
	closeOnce    sync.Once
	wg           sync.WaitGroup
	currentDSN   string
	currentEpoch uint64
	mu           sync.Mutex
}

type SubscriberConfig struct {
	MaxConns int
	MinConns int
	Interval time.Duration
}

func NewSubscriber(rdb redis.UniversalClient, fencing *FencingGate, reconnect ReconnectFunc, cfg SubscriberConfig) *Subscriber {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Second
	}
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 4
	}
	if cfg.MinConns <= 0 {
		cfg.MinConns = 1
	}
	if fencing == nil {
		fencing = NewFencingGate(rdb)
	}
	return &Subscriber{
		rdb:       rdb,
		fencing:   fencing,
		reconnect: reconnect,
		maxConns:  cfg.MaxConns,
		minConns:  cfg.MinConns,
		interval:  cfg.Interval,
		closeCh:   make(chan struct{}),
	}
}

func (s *Subscriber) Fencing() *FencingGate {
	if s == nil {
		return nil
	}
	return s.fencing
}

func (s *Subscriber) Start(ctx context.Context) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runPollLoop(ctx)
	}()
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.runNotifyLoop(ctx)
	}()
}

func (s *Subscriber) Stop() {
	s.closeOnce.Do(func() {
		close(s.closeCh)
	})
	s.wg.Wait()
}

func (s *Subscriber) runPollLoop(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	s.refresh(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closeCh:
			return
		case <-ticker.C:
			s.refresh(ctx)
		}
	}
}

func (s *Subscriber) runNotifyLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closeCh:
			return
		default:
		}
		pubsub := s.rdb.Subscribe(ctx, NotifyChannel())
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				_ = pubsub.Close()
				return
			case <-s.closeCh:
				_ = pubsub.Close()
				return
			case msg, ok := <-ch:
				if !ok {
					_ = pubsub.Close()
					goto resubscribe
				}
				if msg != nil && msg.Payload != "" {
					s.refresh(ctx)
				}
			}
		}
	resubscribe:
		time.Sleep(500 * time.Millisecond)
	}
}

func (s *Subscriber) refresh(ctx context.Context) {
	dsn, epoch, err := ActiveDSN(ctx, s.rdb)
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			slog.Warn("pg failover dsn refresh failed", "error", err)
		}
		return
	}
	if dsn == "" {
		return
	}
	s.mu.Lock()
	unchanged := dsn == s.currentDSN && epoch == s.currentEpoch
	s.mu.Unlock()
	if unchanged {
		return
	}
	if err := s.applyDSN(ctx, dsn, epoch); err != nil {
		slog.Warn("pg failover reconnect failed", "error", err)
	}
}

func (s *Subscriber) applyDSN(ctx context.Context, dsn string, epoch uint64) error {
	pool, err := database.Connect(ctx, dsn, s.maxConns, s.minConns)
	if err != nil {
		return err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return err
	}
	if s.fencing != nil {
		_ = s.fencing.Refresh(ctx)
		s.fencing.AdvanceFloor(epoch)
	}
	if s.reconnect != nil {
		s.reconnect(pool)
	}
	s.mu.Lock()
	s.currentDSN = dsn
	s.currentEpoch = epoch
	s.mu.Unlock()
	slog.Info("pg failover subscriber reconnected", "fencing_epoch", epoch)
	return nil
}

func (s *Subscriber) CurrentDSN() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentDSN
}

func (s *Subscriber) CurrentEpoch() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentEpoch
}
