package logevacuator

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const checkpointFieldCount = 3

type CheckpointRecord struct {
	FileName string
	SHA256   string
}

type CheckpointStore struct {
	path string
}

func NewCheckpointStore(path string) *CheckpointStore {
	return &CheckpointStore{path: path}
}

func (store *CheckpointStore) Load() (CheckpointRecord, error) {
	data, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return CheckpointRecord{}, nil
		}
		return CheckpointRecord{}, err
	}

	line := strings.TrimSpace(string(data))
	if line == "" {
		return CheckpointRecord{}, nil
	}

	fields := strings.Split(line, "|")
	if len(fields) != checkpointFieldCount {
		return CheckpointRecord{}, ErrCheckpointCorrupt
	}

	return CheckpointRecord{
		FileName: fields[0],
		SHA256:   fields[1],
	}, nil
}

func (store *CheckpointStore) Save(record CheckpointRecord) error {
	if err := os.MkdirAll(filepath.Dir(store.path), 0o755); err != nil {
		return err
	}

	tmpPath := store.path + ".tmp"
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	if _, err := writer.WriteString(fmt.Sprintf("%s|%s|1\n", record.FileName, record.SHA256)); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := writer.Flush(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, store.path)
}
