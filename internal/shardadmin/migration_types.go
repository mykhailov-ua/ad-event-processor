package shardadmin

type SlotMigrationDTO struct {
	Version         int32  `json:"version"`
	Slot            int16  `json:"slot"`
	SourceShard     int16  `json:"source_shard"`
	TargetShard     int16  `json:"target_shard"`
	State           string `json:"state"`
	CampaignsTotal  int32  `json:"campaigns_total"`
	CampaignsCopied int32  `json:"campaigns_copied"`
	LastError       string `json:"last_error,omitempty"`
}

type slotMapActivatedAudit struct {
	Version          int32 `json:"version"`
	MigratedSlots    int   `json:"migrated_slots"`
	MigrationCutover bool  `json:"migration_cutover"`
}

type slotMapRollbackAudit struct {
	FromVersion int32 `json:"from_version"`
	ToVersion   int32 `json:"to_version"`
}
