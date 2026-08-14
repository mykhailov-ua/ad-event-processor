package pgfailover

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type StandbyPromoter struct {
	StandbyDSN     string
	PromoteCommand string
	SnapshotSync   bool
	SyncConfig     SnapshotSyncConfig
	PrimaryDSN     string
	MaxConns       int
	MinConns       int
	OnReconnect    func(pool *pgxpool.Pool)
}

func (p *StandbyPromoter) Promote(ctx context.Context) (string, error) {
	if p.StandbyDSN == "" {
		return "", errors.New("standby dsn required")
	}
	if p.SnapshotSync && p.PrimaryDSN != "" {
		if err := p.runSnapshotSync(ctx); err != nil {
			return "", err
		}
	}
	if cmd := strings.TrimSpace(p.PromoteCommand); cmd != "" {
		if err := runPromoteCommand(ctx, cmd); err != nil {
			return "", err
		}
	}
	pool, err := connectStandby(ctx, p.StandbyDSN, p.MaxConns, p.MinConns)
	if err != nil {
		return "", err
	}
	if err := EnsureWritablePrimary(ctx, pool); err != nil {
		pool.Close()
		return "", err
	}
	if p.OnReconnect != nil {
		p.OnReconnect(pool)
	} else {
		pool.Close()
	}
	return p.StandbyDSN, nil
}

func (p *StandbyPromoter) runSnapshotSync(ctx context.Context) error {
	primary, err := connectStandby(ctx, p.PrimaryDSN, 2, 1)
	if err != nil {
		return fmt.Errorf("snapshot primary connect: %w", err)
	}
	defer primary.Close()
	standby, err := connectStandby(ctx, p.StandbyDSN, 2, 1)
	if err != nil {
		return fmt.Errorf("snapshot standby connect: %w", err)
	}
	defer standby.Close()
	return SyncSnapshot(ctx, primary, standby, p.SyncConfig)
}

func runPromoteCommand(ctx context.Context, command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return errors.New("empty promote command")
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("promote command %q: %w: %s", command, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func connectStandby(ctx context.Context, dsn string, maxConns, minConns int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if maxConns <= 0 {
		maxConns = 4
	}
	if minConns <= 0 {
		minConns = 1
	}
	cfg.MaxConns = int32(maxConns)
	cfg.MinConns = int32(minConns)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

// EnsureWritablePrimary blocks until pg_is_in_recovery() is false on the target pool.
// When PgPromoteCommand is empty, operators must promote the standby out of band before
// failover can succeed; otherwise this returns standby not writable after promote timeout.
func EnsureWritablePrimary(ctx context.Context, pool *pgxpool.Pool) error {
	return waitStandbyWritable(ctx, pool)
}

func waitStandbyWritable(ctx context.Context, pool *pgxpool.Pool) error {
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		var inRecovery bool
		err := pool.QueryRow(ctx, `SELECT pg_is_in_recovery()`).Scan(&inRecovery)
		if err == nil && !inRecovery {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return errors.New("standby not writable after promote timeout")
}
