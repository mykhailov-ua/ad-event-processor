package config

import "fmt"

// PostgresPoolBudget returns read+settle connection demand for the shared DB_DSN.
func (c *Config) PostgresPoolBudget() (readMax int, settleMax int) {
	if c == nil {
		return 4, 0
	}
	readMax = c.DBTrackerMaxConns
	if readMax <= 0 {
		readMax = 4
	}
	settleMax = c.PostgresPoolSettleConns(c.SettlementLaneCount())
	return readMax, settleMax
}

func validatePostgresPoolBudget(cfg *Config) error {
	if cfg == nil || cfg.PostgresMaxConnections <= 0 {
		return nil
	}
	return cfg.ValidatePostgresPoolBudget()
}

// ValidatePostgresPoolBudget fails when read+settle pools exceed PG_MAX_CONNECTIONS minus headroom.
func (c *Config) ValidatePostgresPoolBudget() error {
	if c == nil || c.PostgresMaxConnections <= 0 {
		return nil
	}
	readMax, settleMax := c.PostgresPoolBudget()
	total := readMax + settleMax
	headroom := c.PostgresPoolConnHeadroom
	if headroom <= 0 {
		headroom = DefaultPostgresPoolConnHeadroom
	}
	limit := c.PostgresMaxConnections - headroom
	if limit <= 0 {
		return fmt.Errorf("PG_MAX_CONNECTIONS (%d) must exceed PG_POOL_CONN_HEADROOM (%d)", c.PostgresMaxConnections, headroom)
	}
	if total > limit {
		return fmt.Errorf(
			"postgres pool budget exceeded: DB_TRACKER_MAX_CONNS(%d)+settle(%d)=%d > PG_MAX_CONNECTIONS(%d)-headroom(%d)=%d",
			readMax, settleMax, total, c.PostgresMaxConnections, headroom, limit,
		)
	}
	return nil
}
