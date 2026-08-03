package ingestion

import "log/slog"

// BootstrapFromReplica loads the last-known campaign snapshot from disk before Redis/PG.
// PG Sync overwrites when available; this satisfies cold-start without shard 0.
func (r *Registry) BootstrapFromReplica() (int, error) {
	if r == nil {
		return 0, nil
	}
	if len(r.campaignMapSnapshot().byID) > 0 {
		return len(r.campaignMapSnapshot().byID), nil
	}
	loaded, err := r.loadReplica()
	if err != nil {
		return 0, err
	}
	r.storeCampaignSnapshot(loaded)
	n := len(loaded.byID)
	if n > 0 {
		slog.Info("campaign registry bootstrapped from local replica", "campaigns", n)
	}
	return n, nil
}
