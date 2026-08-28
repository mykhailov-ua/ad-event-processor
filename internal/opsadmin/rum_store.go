package opsadmin

import (
	"encoding/json"
	"sync"
	"time"
)

const rumMaxEvents = 200

type ClientRUMEvent struct {
	ReceivedAt string          `json:"received_at"`
	Path       string          `json:"path,omitempty"`
	Vitals     json.RawMessage `json:"vitals,omitempty"`
	API        json.RawMessage `json:"api,omitempty"`
	Guards     json.RawMessage `json:"guards,omitempty"`
	Probes     json.RawMessage `json:"probes,omitempty"`
	Memory     json.RawMessage `json:"memory,omitempty"`
}

type rumStore struct {
	mu     sync.Mutex
	events []ClientRUMEvent
}

var globalRUMStore rumStore

type RUMStoreAdapter struct{}

func NewRUMStoreAdapter() RUMStore {
	return RUMStoreAdapter{}
}

func (a RUMStoreAdapter) AppendClientRUM(ev ClientRUMIngestDTO) {
	appendRUMEvent(ClientRUMEvent{
		Path:   ev.Path,
		Vitals: ev.Vitals,
		API:    ev.API,
		Guards: ev.Guards,
		Probes: ev.Probes,
		Memory: ev.Memory,
	})
}

func (a RUMStoreAdapter) SnapshotClientRUM() []any {
	events := SnapshotRUMEvents()
	out := make([]any, len(events))
	for i := range events {
		out[i] = events[i]
	}
	return out
}

func appendRUMEvent(ev ClientRUMEvent) {
	if ev.ReceivedAt == "" {
		ev.ReceivedAt = time.Now().UTC().Format(time.RFC3339)
	}
	globalRUMStore.mu.Lock()
	defer globalRUMStore.mu.Unlock()
	globalRUMStore.events = append(globalRUMStore.events, ev)
	if len(globalRUMStore.events) > rumMaxEvents {
		globalRUMStore.events = globalRUMStore.events[len(globalRUMStore.events)-rumMaxEvents:]
	}
}

func SnapshotRUMEvents() []ClientRUMEvent {
	globalRUMStore.mu.Lock()
	defer globalRUMStore.mu.Unlock()
	out := make([]ClientRUMEvent, len(globalRUMStore.events))
	copy(out, globalRUMStore.events)
	return out
}
