package fraud

import (
	"log/slog"
	"sync"
)

type ModelRegistry struct {
	mu      sync.RWMutex
	scorers map[string]Scorer
	active  string
}

func NewModelRegistry() *ModelRegistry {
	return &ModelRegistry{
		scorers: make(map[string]Scorer),
	}
}

func (r *ModelRegistry) Register(scorer Scorer) {
	if scorer == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scorers[scorer.Name()] = scorer
	if r.active == "" {
		r.active = scorer.Name()
	}
}

func (r *ModelRegistry) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.scorers[name]; !ok {
		slog.Error("scorer not registered", "name", name)
		return ErrScorerNotRegistered
	}
	r.active = name
	return nil
}

func (r *ModelRegistry) GetActive() Scorer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scorers[r.active]
}

func (r *ModelRegistry) Get(name string) Scorer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.scorers[name]
}
