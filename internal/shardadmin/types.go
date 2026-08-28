package shardadmin

type OutboxHealthSummary struct {
	Pending              int64   `json:"pending"`
	OldestPendingSeconds float64 `json:"oldest_pending_seconds"`
	LastProcessedEventID int64   `json:"last_processed_event_id"`
}

type ShardHealthStatus struct {
	ShardID             int     `json:"shard_id"`
	PingOK              bool    `json:"ping_ok"`
	PingError           string  `json:"ping_error,omitempty"`
	PingLatencyMs       float64 `json:"ping_latency_ms,omitempty"`
	ConfigVersion       *int64  `json:"config_version,omitempty"`
	ConfigVersionLag    int64   `json:"config_version_lag"`
	ConfigVersionSynced bool    `json:"config_version_synced"`
}

type ShardHealthReport struct {
	EmergencyBreaker string              `json:"emergency_breaker"`
	Outbox           OutboxHealthSummary `json:"outbox"`
	Shards           []ShardHealthStatus `json:"shards"`
}

type ShardMetrics struct {
	ShardID   int16
	CPUUsage  float64
	MemoryPct float64
	OpsPerSec int64
	LuaP99Ms  float64
}

type ShardAutoscaleConfig struct {
	Enabled        bool
	CPULimit       float64
	MemoryPctLimit float64
	OpsLimit       int64
	LuaP99Limit    float64
	SlotsToMigrate int16
}

type SlotMapDTO struct {
	Slot    int16  `json:"slot"`
	ShardID int16  `json:"shard_id"`
	State   string `json:"state"`
}

type SlotMapVersionDTO struct {
	Version        int32        `json:"version"`
	ActiveVersion  int32        `json:"active_version"`
	SlotCount      int32        `json:"slot_count"`
	MigratingCount int          `json:"migrating_count"`
	Slots          []SlotMapDTO `json:"slots,omitempty"`
}
