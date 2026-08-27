package logpipeline

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CheckpointRecord struct {
	SourceKey     string    `json:"source_key"`
	DestKey       string    `json:"dest_key"`
	SourceSHA256  string    `json:"source_sha256"`
	DestSHA256    string    `json:"dest_sha256"`
	OriginalCount int64     `json:"original_count"`
	KeptCount     int64     `json:"kept_count"`
	CompactedAt   time.Time `json:"compacted_at"`
}

type CheckpointStore struct {
	path     string
	mu       sync.Mutex
	bySource map[string]CheckpointRecord
}

func NewCheckpointStore(path string) *CheckpointStore {
	return &CheckpointStore{
		path:     path,
		bySource: make(map[string]CheckpointRecord),
	}
}

func (st *CheckpointStore) Load() error {
	st.mu.Lock()
	defer st.mu.Unlock()

	data, err := os.ReadFile(st.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	st.bySource = make(map[string]CheckpointRecord)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var record CheckpointRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return fmt.Errorf("%w at line %d", ErrCheckpointCorrupt, lineNo)
		}
		if record.SourceKey == "" {
			continue
		}
		st.bySource[record.SourceKey] = record
	}
	return scanner.Err()
}

func (st *CheckpointStore) IsCompacted(sourceKey, sourceSHA256 string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	record, ok := st.bySource[sourceKey]
	if !ok {
		return false
	}
	if sourceSHA256 == "" {
		return true
	}
	return record.SourceSHA256 == sourceSHA256
}

func (st *CheckpointStore) Has(sourceKey string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	_, ok := st.bySource[sourceKey]
	return ok
}

func (st *CheckpointStore) Count() int {
	st.mu.Lock()
	defer st.mu.Unlock()
	return len(st.bySource)
}

func (st *CheckpointStore) Get(sourceKey string) (CheckpointRecord, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()
	record, ok := st.bySource[sourceKey]
	return record, ok
}

func (st *CheckpointStore) Save(record CheckpointRecord) error {
	st.mu.Lock()
	defer st.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(st.path), 0o755); err != nil {
		return err
	}

	line, err := json.Marshal(record)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(st.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}

	st.bySource[record.SourceKey] = record
	return nil
}
