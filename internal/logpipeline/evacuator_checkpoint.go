package logpipeline

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const checkpointFieldCount = 3

type EvacuatorCheckpointRecord struct {
	FileName string
	SHA256   string
}

type EvacuatorCheckpointStore struct {
	path string
}

func NewEvacuatorCheckpointStore(path string) *EvacuatorCheckpointStore {
	return &EvacuatorCheckpointStore{path: path}
}

func (store *EvacuatorCheckpointStore) Load() (EvacuatorCheckpointRecord, error) {
	data, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return EvacuatorCheckpointRecord{}, nil
		}
		return EvacuatorCheckpointRecord{}, err
	}

	line := strings.TrimSpace(string(data))
	if line == "" {
		return EvacuatorCheckpointRecord{}, nil
	}

	fields := strings.Split(line, "|")
	if len(fields) != checkpointFieldCount {
		return EvacuatorCheckpointRecord{}, ErrEvacuatorCheckpointCorrupt
	}

	return EvacuatorCheckpointRecord{
		FileName: fields[0],
		SHA256:   fields[1],
	}, nil
}

func (store *EvacuatorCheckpointStore) Save(record EvacuatorCheckpointRecord) error {
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
