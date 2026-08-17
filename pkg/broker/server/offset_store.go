package server

import (
	"context"
	"errors"
	"sync"
)

type OffsetStore interface {
	Commit(ctx context.Context, topic, group string, offset uint64) (uint64, error)
	Committed(ctx context.Context, topic, group string) (uint64, error)
	MinCommitted(ctx context.Context, topic string) (uint64, bool, error)
	ListGroups(ctx context.Context, topic string) (map[string]uint64, error)
}

type MemoryOffsetStore struct {
	mu      sync.RWMutex
	byTopic map[string]map[string]uint64
}

func NewMemoryOffsetStore() *MemoryOffsetStore {
	return &MemoryOffsetStore{
		byTopic: make(map[string]map[string]uint64),
	}
}

func (s *MemoryOffsetStore) Commit(_ context.Context, topic, group string, offset uint64) (uint64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	groups := s.byTopic[topic]
	if groups == nil {
		groups = make(map[string]uint64)
		s.byTopic[topic] = groups
	}
	if cur, ok := groups[group]; ok && offset <= cur {
		return cur, nil
	}
	groups[group] = offset
	return offset, nil
}

func (s *MemoryOffsetStore) Committed(_ context.Context, topic, group string) (uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if groups, ok := s.byTopic[topic]; ok {
		return groups[group], nil
	}
	return 0, nil
}

func (s *MemoryOffsetStore) MinCommitted(_ context.Context, topic string) (uint64, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	groups, ok := s.byTopic[topic]
	if !ok || len(groups) == 0 {
		return 0, false, nil
	}
	var minOffset uint64
	first := true
	for _, off := range groups {
		if first || off < minOffset {
			minOffset = off
			first = false
		}
	}
	return minOffset, true, nil
}

func (s *MemoryOffsetStore) ListGroups(_ context.Context, topic string) (map[string]uint64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	src, ok := s.byTopic[topic]
	if !ok || len(src) == 0 {
		return nil, nil
	}
	out := make(map[string]uint64, len(src))
	for group, off := range src {
		out[group] = off
	}
	return out, nil
}

func validateOffsetKey(topic, group string) error {
	if err := validateTopicNameForOffset(topic); err != nil {
		return err
	}
	if err := validateGroupNameForOffset(group); err != nil {
		return err
	}
	return nil
}

func validateTopicNameForOffset(topic string) error {
	if topic == "" {
		return errors.New("topic name is empty")
	}
	if len(topic) > 255 {
		return errors.New("topic name too long")
	}
	return nil
}

func validateGroupNameForOffset(group string) error {
	if group == "" {
		return errors.New("consumer group is empty")
	}
	if len(group) > 255 {
		return errors.New("consumer group name too long")
	}
	return nil
}
