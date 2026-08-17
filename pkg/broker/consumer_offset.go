// Package broker provides the log broker client and server protocol.
package broker

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bidshard/ad-event-processor/pkg/broker/protocol"
)

type ConsumerOffsetTracker struct {
	mu      sync.RWMutex
	dir     string
	offsets map[string]uint64
}

func NewConsumerOffsetTracker(dataDir string) (*ConsumerOffsetTracker, error) {
	if dataDir == "" {
		dataDir = "var/lib/bidshard/broker/offsets"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create consumer offset dir: %w", err)
	}
	t := &ConsumerOffsetTracker{
		dir:     dataDir,
		offsets: make(map[string]uint64),
	}
	if err := t.LoadAll(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *ConsumerOffsetTracker) key(topic string, partition uint16, group string) string {
	return fmt.Sprintf("%s:%d:%s", topic, partition, group)
}

func (t *ConsumerOffsetTracker) filename(topic string, partition uint16, group string) string {
	safeTopic := sanitizeFilename(topic)
	safeGroup := sanitizeFilename(group)
	return filepath.Join(t.dir, fmt.Sprintf("%s_%d_%s.offset", safeTopic, partition, safeGroup))
}

func sanitizeFilename(s string) string {
	buf := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' {
			buf[i] = c
		} else {
			buf[i] = '_'
		}
	}
	return string(buf)
}

func (t *ConsumerOffsetTracker) CommitOffset(topic string, partition uint16, group string, offset uint64) error {
	if err := protocol.ValidateConsumerGroup(group); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	k := t.key(topic, partition, group)
	if cur, ok := t.offsets[k]; ok && offset <= cur {
		return nil
	}

	fn := t.filename(topic, partition, group)
	tmpFn := fn + ".tmp"

	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], offset)

	if err := os.WriteFile(tmpFn, buf[:], 0o644); err != nil {
		return fmt.Errorf("write temp offset file: %w", err)
	}
	if err := os.Rename(tmpFn, fn); err != nil {
		return fmt.Errorf("rename offset file: %w", err)
	}

	t.offsets[k] = offset
	return nil
}

func (t *ConsumerOffsetTracker) GetCommittedOffset(topic string, partition uint16, group string) (uint64, error) {
	t.mu.RLock()
	k := t.key(topic, partition, group)
	val, ok := t.offsets[k]
	t.mu.RUnlock()

	if ok {
		return val, nil
	}

	fn := t.filename(topic, partition, group)
	data, err := os.ReadFile(fn)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if len(data) < 8 {
		return 0, nil
	}

	off := binary.BigEndian.Uint64(data[:8])

	t.mu.Lock()
	t.offsets[k] = off
	t.mu.Unlock()

	return off, nil
}

func (t *ConsumerOffsetTracker) LoadAll() error {
	entries, err := os.ReadDir(t.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".offset" {
			continue
		}
		fullPath := filepath.Join(t.dir, entry.Name())
		data, readErr := os.ReadFile(fullPath)
		if readErr != nil || len(data) < 8 {
			continue
		}
		off := binary.BigEndian.Uint64(data[:8])

		stem := entry.Name()[:len(entry.Name())-7]
		t.offsets[stem] = off
	}
	return nil
}
