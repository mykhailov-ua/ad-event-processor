package governance

type auditTxSourceMeta struct {
	TxSource string `json:"tx_source"`
}

type auditQuotaRepairMeta struct {
	OutboxEventID int64  `json:"outbox_event_id"`
	Reason        string `json:"reason"`
	RepairMicro   int64  `json:"repair_micro"`
}

type auditQuotaDeadShardRelease struct {
	ShardID      int   `json:"shard_id"`
	RowsReleased int64 `json:"rows_released"`
}
