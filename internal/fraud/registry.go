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

func (modelRegistry *ModelRegistry) Register(scorer Scorer) {
	if scorer == nil {
		return
	}
	modelRegistry.mu.Lock()
	defer modelRegistry.mu.Unlock()
	modelRegistry.scorers[scorer.Name()] = scorer
	if modelRegistry.active == "" {
		modelRegistry.active = scorer.Name()
	}
}

func (modelRegistry *ModelRegistry) SetActive(name string) error {
	modelRegistry.mu.Lock()
	defer modelRegistry.mu.Unlock()
	if _, ok := modelRegistry.scorers[name]; !ok {
		slog.Error("scorer not registered", "name", name)
		return ErrScorerNotRegistered
	}
	modelRegistry.active = name
	return nil
}

func (modelRegistry *ModelRegistry) GetActive() Scorer {
	modelRegistry.mu.RLock()
	defer modelRegistry.mu.RUnlock()
	return modelRegistry.scorers[modelRegistry.active]
}

func (modelRegistry *ModelRegistry) Get(name string) Scorer {
	modelRegistry.mu.RLock()
	defer modelRegistry.mu.RUnlock()
	return modelRegistry.scorers[name]
}
