package inbound

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store caches per-customer inbound auth config refreshed from Postgres.
type Store struct {
	pool     *pgxpool.Pool
	interval time.Duration
	snapshot atomic.Value
}

func NewStore(pool *pgxpool.Pool, interval time.Duration) *Store {
	if interval <= 0 {
		interval = 60 * time.Second
	}
	st := &Store{pool: pool, interval: interval}
	st.snapshot.Store(map[uuid.UUID]AuthConfig{})
	return st
}

func (st *Store) Start(ctx context.Context) {
	if st == nil || st.pool == nil {
		return
	}
	ticker := time.NewTicker(st.interval)
	defer ticker.Stop()
	for {
		if err := st.refresh(ctx); err != nil {
			_ = err
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (st *Store) refresh(ctx context.Context) error {
	rows, err := st.pool.Query(ctx, `
		SELECT id, postback_inbound_ip_allowlist, postback_inbound_secret_encrypted
		FROM customers
		WHERE cardinality(postback_inbound_ip_allowlist) > 0
		   OR postback_inbound_secret_encrypted IS NOT NULL`)
	if err != nil {
		return err
	}
	defer rows.Close()
	next := make(map[uuid.UUID]AuthConfig)
	for rows.Next() {
		var id uuid.UUID
		var allowlist []string
		var secret []byte
		if err := rows.Scan(&id, &allowlist, &secret); err != nil {
			return err
		}
		next[id] = AuthConfig{IPAllowlist: allowlist, Secret: secret}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	st.snapshot.Store(next)
	return nil
}

func (st *Store) Config(customerID uuid.UUID) AuthConfig {
	if st == nil {
		return AuthConfig{}
	}
	m, _ := st.snapshot.Load().(map[uuid.UUID]AuthConfig)
	if m == nil {
		return AuthConfig{}
	}
	return m[customerID]
}
