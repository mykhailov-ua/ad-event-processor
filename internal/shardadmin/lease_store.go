package shardadmin

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

func NewLeaseFencingRegistry(baseDir string) (*LeaseFencingRegistry, error) {
	if baseDir == "" {
		return nil, errors.New("lease fencing dir required")
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &LeaseFencingRegistry{
		baseDir: baseDir,
		stores:  make(map[uuid.UUID]*LeaseFencingStore),
	}, nil
}

func (r *LeaseFencingRegistry) storeFor(replicaSetID uuid.UUID) (*LeaseFencingStore, error) {
	if r == nil {
		return nil, errors.New("lease fencing registry unavailable")
	}
	if replicaSetID == uuid.Nil {
		return nil, errors.New("replica set id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.stores[replicaSetID]; ok {
		return s, nil
	}
	dir := filepath.Join(r.baseDir, replicaSetID.String())
	s, err := NewLeaseFencingStore(dir)
	if err != nil {
		return nil, err
	}
	r.stores[replicaSetID] = s
	return s, nil
}

func (r *LeaseFencingRegistry) Next(replicaSetID uuid.UUID) (int64, error) {
	s, err := r.storeFor(replicaSetID)
	if err != nil {
		return 0, err
	}
	return s.Next()
}

func (r *LeaseFencingRegistry) Validate(replicaSetID uuid.UUID, epoch int64) error {
	s, err := r.storeFor(replicaSetID)
	if err != nil {
		return err
	}
	return s.Validate(epoch)
}

func NewLeaseFencingStore(dir string) (*LeaseFencingStore, error) {
	if dir == "" {
		return nil, errors.New("lease fencing dir required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &LeaseFencingStore{dir: dir}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *LeaseFencingStore) Floor() uint64 {
	if s == nil {
		return 0
	}
	return s.epoch.Load()
}

func (s *LeaseFencingStore) Next() (int64, error) {
	if s == nil {
		return 1, nil
	}
	next := s.epoch.Add(1)
	if err := s.persist(next); err != nil {
		return 0, err
	}
	return int64(next), nil
}

func (s *LeaseFencingStore) Validate(epoch int64) error {
	if s == nil {
		return nil
	}
	floor := s.Floor()
	if floor == 0 {
		return nil
	}
	if epoch <= 0 || uint64(epoch) < floor {
		return ErrStaleFencingEpoch
	}
	return nil
}

func (s *LeaseFencingStore) load() error {
	path := filepath.Join(s.dir, leaseFencingEpochFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) < 8 {
		return nil
	}
	s.epoch.Store(binary.BigEndian.Uint64(data[:8]))
	return nil
}

// persist: write+rename epoch file on local disk; broker AppendFenced rejects epochs below Floor().
func (s *LeaseFencingStore) persist(epoch uint64) error {
	path := filepath.Join(s.dir, leaseFencingEpochFile)
	tmp := path + ".tmp"
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], epoch)
	if err := os.WriteFile(tmp, buf[:], 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
