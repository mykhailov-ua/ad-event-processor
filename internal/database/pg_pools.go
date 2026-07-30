package database

import (
	"context"
	"fmt"

	"espx/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgPools struct {
	Read   *pgxpool.Pool
	Settle *pgxpool.Pool
}

func ConnectPgPools(ctx context.Context, cfg *config.Config) (*PgPools, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	lanes := cfg.SettlementLaneCount()
	settleMax := cfg.PgPoolSettleConns(lanes)
	readMax := cfg.DBTrackerMaxConns
	if readMax <= 0 {
		readMax = 4
	}
	readPool, err := Connect(ctx, string(cfg.DBDSN), readMax, cfg.DBMinConns)
	if err != nil {
		return nil, fmt.Errorf("read pool: %w", err)
	}
	settlePool, err := Connect(ctx, string(cfg.DBDSN), settleMax, 1)
	if err != nil {
		readPool.Close()
		return nil, fmt.Errorf("settle pool: %w", err)
	}
	return &PgPools{Read: readPool, Settle: settlePool}, nil
}

func (p *PgPools) Close() {
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
