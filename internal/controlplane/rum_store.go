package controlplane

import (
	"encoding/json"
	"sync"
	"time"
)

const rumMaxEvents = 200

// ClientRUMEvent is a sampled admin UI telemetry batch from the browser.
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

func snapshotRUMEvents() []ClientRUMEvent {
	globalRUMStore.mu.Lock()
	defer globalRUMStore.mu.Unlock()
	out := make([]ClientRUMEvent, len(globalRUMStore.events))
	copy(out, globalRUMStore.events)
	return out
}

func resetRUMStoreForTest() {
	globalRUMStore.mu.Lock()
	defer globalRUMStore.mu.Unlock()
	globalRUMStore.events = nil
}
