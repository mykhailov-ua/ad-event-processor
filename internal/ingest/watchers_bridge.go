package ingest

import (
	"ad-event-processor/internal/ingest/watchers"
)

type (
	CampaignUpdateWatcherConfig = watchers.CampaignUpdateWatcherConfig
	CampaignUpdateWatcher       = watchers.CampaignUpdateWatcher
	SlotMapWatcherConfig        = watchers.SlotMapWatcherConfig
	SlotMapWatcher              = watchers.SlotMapWatcher
)

var (
	NewCampaignUpdateWatcher = watchers.NewCampaignUpdateWatcher
	NewSlotMapWatcher        = watchers.NewSlotMapWatcher
)
