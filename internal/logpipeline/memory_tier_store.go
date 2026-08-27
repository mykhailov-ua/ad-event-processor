package logpipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryObjectStore struct {
	mu      sync.RWMutex
	objects map[string]memoryObject
}

type memoryObject struct {
	data     []byte
	modTime  time.Time
	metadata map[string]string
}

func NewMemoryObjectStore() *MemoryObjectStore {
	return &MemoryObjectStore{objects: make(map[string]memoryObject)}
}

func (st *MemoryObjectStore) Put(key string, data []byte, modTime time.Time, metadata map[string]string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	metaCopy := make(map[string]string, len(metadata))
	for k, v := range metadata {
		metaCopy[k] = v
	}
	st.objects[key] = memoryObject{
		data:     append([]byte(nil), data...),
		modTime:  modTime,
		metadata: metaCopy,
	}
}

func (st *MemoryObjectStore) Get(key string) ([]byte, bool) {
	st.mu.RLock()
	defer st.mu.RUnlock()
	object, ok := st.objects[key]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), object.data...), true
}

func (st *MemoryObjectStore) Delete(key string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.objects, key)
}

func (st *MemoryObjectStore) List(prefix string) []memoryListedObject {
	st.mu.RLock()
	defer st.mu.RUnlock()
	var objects []memoryListedObject
	for key, object := range st.objects {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		objects = append(objects, memoryListedObject{
			key:     key,
			modTime: object.modTime,
			size:    int64(len(object.data)),
		})
	}
	sort.Slice(objects, func(i, j int) bool {
		return objects[i].key < objects[j].key
	})
	return objects
}

type memoryListedObject struct {
	key     string
	modTime time.Time
	size    int64
}

type MemoryS3TierStore struct {
	hotPrefix  string
	warmPrefix string
	mem        *MemoryObjectStore
	local      *LocalTierStore
}

func NewMemoryS3TierStore(scratchDir, hotPrefix, warmPrefix string, mem *MemoryObjectStore) *MemoryS3TierStore {
	if mem == nil {
		mem = NewMemoryObjectStore()
	}
	if hotPrefix != "" && !strings.HasSuffix(hotPrefix, "/") {
		hotPrefix += "/"
	}
	if warmPrefix != "" && !strings.HasSuffix(warmPrefix, "/") {
		warmPrefix += "/"
	}
	hotDir := filepath.Join(scratchDir, "hot")
	warmDir := filepath.Join(scratchDir, "warm")
	_ = os.MkdirAll(hotDir, 0o755)
	_ = os.MkdirAll(warmDir, 0o755)
	return &MemoryS3TierStore{
		hotPrefix:  hotPrefix,
		warmPrefix: warmPrefix,
		mem:        mem,
		local:      NewLocalTierStore(hotDir, warmDir),
	}
}

func (st *MemoryS3TierStore) ListHot(ctx context.Context, olderThan time.Time) ([]TierObject, error) {
	for _, object := range st.mem.List(st.hotPrefix) {
		name := strings.TrimPrefix(object.key, st.hotPrefix)
		if name == "" || !isHotSegmentName(filepath.Base(name)) {
			continue
		}
		if !object.modTime.Before(olderThan) {
			continue
		}
		localPath := filepath.Join(st.local.SourceDir, filepath.Base(name))
		if _, err := os.Stat(localPath); err == nil {
			continue
		}
		data, ok := st.mem.Get(object.key)
		if !ok {
			continue
		}
		if err := os.WriteFile(localPath, data, 0o644); err != nil {
			return nil, err
		}
		if err := os.Chtimes(localPath, object.modTime, object.modTime); err != nil {
			return nil, err
		}
	}
	return st.local.ListHot(ctx, olderThan)
}

func (st *MemoryS3TierStore) uploadWarmArtifacts(destKey, sha256 string) error {
	warmPath := filepath.Join(st.local.WarmDir, destKey)
	data, err := os.ReadFile(warmPath)
	if err != nil {
		return err
	}
	st.mem.Put(st.warmPrefix+destKey, data, time.Now().UTC(), map[string]string{
		s3MetadataSHA256Key: sha256,
	})
	metaPath := strings.TrimSuffix(warmPath, ".zst") + ".meta.json"
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}
	metaDigest, err := computeFileDigest(metaPath)
	if err != nil {
		return err
	}
	metaKey := st.warmPrefix + strings.TrimSuffix(destKey, ".zst") + ".meta.json"
	st.mem.Put(metaKey, metaBytes, time.Now().UTC(), map[string]string{
		s3MetadataSHA256Key: metaDigest.SHA256,
	})
	return nil
}

func (st *MemoryS3TierStore) WriteWarm(ctx context.Context, destKey string, plaintext []byte, meta CompactionMeta) error {
	if err := st.local.WriteWarm(ctx, destKey, plaintext, meta); err != nil {
		return err
	}
	return st.uploadWarmArtifacts(destKey, meta.DestSHA256)
}

func (st *MemoryS3TierStore) WriteWarmFromFile(ctx context.Context, destKey, filteredPath string, meta CompactionMeta) (string, error) {
	destSHA, err := st.local.WriteWarmFromFile(ctx, destKey, filteredPath, meta)
	if err != nil {
		return "", err
	}
	if err := st.uploadWarmArtifacts(destKey, destSHA); err != nil {
		st.local.RemoveWarmArtifacts(destKey)
		return "", err
	}
	return destSHA, nil
}

func (st *MemoryS3TierStore) RemoveHot(ctx context.Context, obj TierObject) error {
	if err := st.local.RemoveHot(ctx, obj); err != nil {
		return err
	}
	st.mem.Delete(st.hotPrefix + hotKeyFromCompacting(obj.Key))
	return nil
}

func (st *MemoryS3TierStore) ClaimHot(ctx context.Context, obj TierObject) (TierObject, error) {
	return st.local.ClaimHot(ctx, obj)
}

func (st *MemoryS3TierStore) RollbackHot(ctx context.Context, obj TierObject) error {
	return st.local.RollbackHot(ctx, obj)
}

func (st *MemoryS3TierStore) ListStuckCompacting(ctx context.Context) ([]TierObject, error) {
	return st.local.ListStuckCompacting(ctx)
}

func (st *MemoryS3TierStore) RemoveCompacting(ctx context.Context, obj TierObject) error {
	hotKey := hotKeyFromCompacting(obj.Key)
	if err := st.local.RemoveCompacting(ctx, obj); err != nil {
		return err
	}
	st.mem.Delete(st.hotPrefix + hotKey)
	return nil
}

func (st *MemoryS3TierStore) RemoveWarmArtifacts(destKey string) {
	st.local.RemoveWarmArtifacts(destKey)
}

func (st *MemoryS3TierStore) SeedHot(name string, data []byte, modTime time.Time) {
	st.mem.Put(st.hotPrefix+name, data, modTime, nil)
}

func (st *MemoryS3TierStore) WarmObjectCount() int {
	count := 0
	for _, object := range st.mem.List(st.warmPrefix) {
		if strings.HasSuffix(object.key, ".compact.zst") {
			count++
		}
	}
	return count
}

func (st *MemoryS3TierStore) WarmObject(destKey string) ([]byte, bool) {
	return st.mem.Get(st.warmPrefix + destKey)
}

func (st *MemoryS3TierStore) HotObjectCount() int {
	return len(st.mem.List(st.hotPrefix))
}

func (st *MemoryS3TierStore) String() string {
	return fmt.Sprintf("memory-s3 hot=%d warm=%d", st.HotObjectCount(), st.WarmObjectCount())
}
