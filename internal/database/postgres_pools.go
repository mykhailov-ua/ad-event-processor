package database

import (
	"context"
	"fmt"

	"ad-event-processor/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresPools: separate pgx pools so settlement lanes cannot exhaust tracker read conns.
type PostgresPools struct {
	Read   *pgxpool.Pool
	Settle *pgxpool.Pool
}

func ConnectPostgresPools(ctx context.Context, cfg *config.Config) (*PostgresPools, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	lanes := cfg.SettlementLaneCount()
	settleMax := cfg.PostgresPoolSettleConns(lanes)
	readMax := cfg.DBTrackerMaxConns
	if readMax <= 0 {
		readMax = 4
	}
	readPool, err := Connect(ctx, string(cfg.DBDSN), readMax, cfg.DBMinConns)
	if err != nil {
		return nil, fmt.Errorf("read pool: %w", err)
	}
	settlementPool, err := Connect(ctx, string(cfg.DBDSN), settleMax, 1)
	if err != nil {
		readPool.Close()
		return nil, fmt.Errorf("settle pool: %w", err)
	}
	return &PostgresPools{Read: readPool, Settle: settlementPool}, nil
}

func (p *PostgresPools) Close() {
	if p == nil {
		return
	}
	if p.Read != nil {
		p.Read.Close()
	}
	if p.Settle != nil && p.Settle != p.Read {
		p.Settle.Close()
	}
}
