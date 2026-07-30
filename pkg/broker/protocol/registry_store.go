package protocol

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const registryFileVersion = 1

type RegistrySnapshot struct {
	Version uint32            `json:"version"`
	Topics  map[string]uint16 `json:"topics"`
	NextID  uint32            `json:"next_id"`
}

type FileRegistryStore struct {
	path string
}

func NewFileRegistryStore(dataDir string) (*FileRegistryStore, error) {
	dir := filepath.Join(dataDir, ".topics")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create topics dir: %w", err)
	}
	return &FileRegistryStore{path: filepath.Join(dir, "registry.json")}, nil
}

func (s *FileRegistryStore) Load() (RegistrySnapshot, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return RegistrySnapshot{Version: registryFileVersion, Topics: make(map[string]uint16)}, nil
		}
		return RegistrySnapshot{}, err
	}
	var snap RegistrySnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return RegistrySnapshot{}, fmt.Errorf("decode registry: %w", err)
	}
	if snap.Topics == nil {
		snap.Topics = make(map[string]uint16)
	}
	if snap.Version == 0 {
		snap.Version = registryFileVersion
	}
	return snap, nil
}

func (s *FileRegistryStore) Save(snap RegistrySnapshot) error {
	if snap.Topics == nil {
		snap.Topics = make(map[string]uint16)
	}
	if snap.Version == 0 {
		snap.Version = registryFileVersion
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *FileRegistryStore) Path() string {
	return s.path
}

func validateTopicName(name string) error {
	if name == "" {
		return errors.New("topic name is empty")
	}
	if len(name) > 255 {
		return errors.New("topic name too long")
	}
	return nil
}

func ValidateTopicNameForStore(name string) error {
	return validateTopicName(name)
}

func validateTopicID(id uint16) error {
	if id == 0 {
		return errors.New("topic id must be non-zero")
	}
	return nil
}
