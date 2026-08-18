package controlplane

type rumStoreAdapter struct{}

func (rumStoreAdapter) AppendClientRUM(ev ClientRUMIngestDTO) {
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
