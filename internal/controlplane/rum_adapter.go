package controlplane

import "github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"

type rumStoreAdapter struct{}

func (rumStoreAdapter) AppendClientRUM(ev adminapi.ClientRUMIngestDTO) {
	appendRUMEvent(ClientRUMEvent{
		Path:   ev.Path,
		Vitals: ev.Vitals,
		API:    ev.API,
		Guards: ev.Guards,
		Probes: ev.Probes,
		Memory: ev.Memory,
	})
}

func (rumStoreAdapter) SnapshotClientRUM() []any {
	events := snapshotRUMEvents()
	out := make([]any, len(events))
	for i := range events {
		out[i] = events[i]
	}
	return out
}
