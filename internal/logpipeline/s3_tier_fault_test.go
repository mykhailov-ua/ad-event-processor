package logpipeline

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type faultMemoryWarmStore struct {
	inner         *MemoryS3TierStore
	failWarmWrite atomic.Uint32
}

func (st *faultMemoryWarmStore) ListHot(ctx context.Context, olderThan time.Time) ([]TierObject, error) {
	return st.inner.ListHot(ctx, olderThan)
}

func (st *faultMemoryWarmStore) WriteWarm(ctx context.Context, destKey string, plaintext []byte, meta CompactionMeta) error {
	return st.inner.WriteWarm(ctx, destKey, plaintext, meta)
}

func (st *faultMemoryWarmStore) RemoveHot(ctx context.Context, obj TierObject) error {
	return st.inner.RemoveHot(ctx, obj)
}

func (st *faultMemoryWarmStore) ClaimHot(ctx context.Context, obj TierObject) (TierObject, error) {
	return st.inner.ClaimHot(ctx, obj)
}

func (st *faultMemoryWarmStore) RollbackHot(ctx context.Context, obj TierObject) error {
	return st.inner.RollbackHot(ctx, obj)
}

func (st *faultMemoryWarmStore) ListStuckCompacting(ctx context.Context) ([]TierObject, error) {
	return st.inner.ListStuckCompacting(ctx)
}

func (st *faultMemoryWarmStore) RemoveCompacting(ctx context.Context, obj TierObject) error {
	return st.inner.RemoveCompacting(ctx, obj)
}

func (st *faultMemoryWarmStore) WriteWarmFromFile(ctx context.Context, destKey, filteredPath string, meta CompactionMeta) (string, error) {
	if st.failWarmWrite.Load() > 0 {
		st.failWarmWrite.Add(^uint32(0))
		return "", errors.New("injected warm upload failure")
	}
	return st.inner.WriteWarmFromFile(ctx, destKey, filteredPath, meta)
}

func (st *faultMemoryWarmStore) RemoveWarmArtifacts(destKey string) {
	st.inner.RemoveWarmArtifacts(destKey)
}

func TestFault_logCompactorS3TierExactlyOnce(t *testing.T) {
	scratchDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")
	mem := NewMemoryObjectStore()
	store := NewMemoryS3TierStore(scratchDir, "hot", "warm", mem)

	oldTime := time.Now().Add(-48 * time.Hour)
	store.SeedHot("segment_s3.log", buildSegmentPayload(t, 32, 1), oldTime)

	compactor := NewCompactor(CompactorConfig{
		HotMinAge: 0,
		WarmDir:   store.local.WarmDir,
		SourceDir: store.local.SourceDir,
	}, store, NewCheckpointStore(checkpointPath), nil)

	require.NoError(t, compactor.RunOnce(context.Background()))
	require.NoError(t, compactor.RunOnce(context.Background()))

	assert.Equal(t, 1, store.HotObjectCount())
	assert.Equal(t, 1, store.WarmObjectCount())

	destKey := warmDestKey("segment_s3.log")
	warmData, ok := store.WarmObject(destKey)
	require.True(t, ok)
	assert.NotEmpty(t, warmData)

	faultproof.Log(t, "log_compactor_s3_tier_exactly_once", map[string]string{
		"subsystem":     "log_compactor",
		"warm_uploaded": "true",
		"hot_removed":   "false",
		"exactly_once":  "true",
	})
}

func TestFault_logCompactorS3WarmUploadRetry(t *testing.T) {
	scratchDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")
	mem := NewMemoryObjectStore()
	inner := NewMemoryS3TierStore(scratchDir, "hot", "warm", mem)
	store := &faultMemoryWarmStore{inner: inner}
	store.failWarmWrite.Store(1)

	oldTime := time.Now().Add(-48 * time.Hour)
	inner.SeedHot("segment_retry.log", buildSegmentPayload(t, 8, 1), oldTime)

	compactor := NewCompactor(CompactorConfig{
		HotMinAge: 0,
		WarmDir:   inner.local.WarmDir,
		SourceDir: inner.local.SourceDir,
	}, store, NewCheckpointStore(checkpointPath), nil)

	require.ErrorIs(t, compactor.RunOnce(context.Background()), ErrCompactionFailures)
	assert.Equal(t, 0, inner.WarmObjectCount())

	checkpoint := NewCheckpointStore(checkpointPath)
	_, ok := checkpoint.Get("segment_retry.log")
	require.False(t, ok)

	require.NoError(t, compactor.RunOnce(context.Background()))
	assert.Equal(t, 1, inner.WarmObjectCount())

	faultproof.Log(t, "log_compactor_s3_warm_upload_retry", map[string]string{
		"subsystem":        "log_compactor",
		"retry_succeeded":  "true",
		"checkpoint_clean": "true",
	})
}

func TestFault_logCompactorLeaderElectionSingleWriter(t *testing.T) {
	sourceDir := t.TempDir()
	warmDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")
	lockPath := filepath.Join(t.TempDir(), "leader.lock")

	for i := range 10 {
		writeHotSegment(t, sourceDir, "segment_leader_"+itoa(i)+".log", buildSegmentPayload(t, 4, 0))
	}

	store := NewLocalTierStore(sourceDir, warmDir)
	checkpoint := NewCheckpointStore(checkpointPath)

	lockA := NewFileLeaderLock(lockPath)
	lockB := NewFileLeaderLock(lockPath)
	compactorA := NewCompactor(CompactorConfig{
		HotMinAge:    0,
		WarmDir:      warmDir,
		SourceDir:    sourceDir,
		WorkInterval: 10 * time.Millisecond,
	}, store, checkpoint, nil, WithLeaderLock(lockA))
	compactorB := NewCompactor(CompactorConfig{
		HotMinAge:    0,
		WarmDir:      warmDir,
		SourceDir:    sourceDir,
		WorkInterval: 10 * time.Millisecond,
	}, store, checkpoint, nil, WithLeaderLock(lockB))

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = compactorA.Run(ctx)
	}()
	go func() {
		defer wg.Done()
		_ = compactorB.Run(ctx)
	}()
	wg.Wait()

	require.Equal(t, 10, checkpoint.Count())

	faultproof.Log(t, "log_compactor_leader_election", map[string]string{
		"subsystem":  "log_compactor",
		"instances":  "2",
		"segments":   "10",
		"duplicates": "0",
	})
}

func TestFileLeaderLock_exclusive(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "leader.lock")
	lockA := NewFileLeaderLock(lockPath)
	lockB := NewFileLeaderLock(lockPath)

	acquiredA, err := lockA.TryAcquire()
	require.NoError(t, err)
	require.True(t, acquiredA)

	acquiredB, err := lockB.TryAcquire()
	require.NoError(t, err)
	require.False(t, acquiredB)

	require.NoError(t, lockA.Release())

	acquiredB, err = lockB.TryAcquire()
	require.NoError(t, err)
	require.True(t, acquiredB)
	require.NoError(t, lockB.Release())
}
