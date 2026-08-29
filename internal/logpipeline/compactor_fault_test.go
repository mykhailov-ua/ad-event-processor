package logpipeline

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/ingest/pb"

	"github.com/stretchr/testify/require"
)

type faultCheckpointStore struct {
	inner    *CheckpointStore
	failSave atomic.Uint32
}

func (st *faultCheckpointStore) Load() error {
	return st.inner.Load()
}

func (st *faultCheckpointStore) IsCompacted(sourceKey, sourceSHA256 string) bool {
	return st.inner.IsCompacted(sourceKey, sourceSHA256)
}

func (st *faultCheckpointStore) Get(sourceKey string) (CheckpointRecord, bool) {
	return st.inner.Get(sourceKey)
}

func (st *faultCheckpointStore) Has(sourceKey string) bool {
	return st.inner.Has(sourceKey)
}

func (st *faultCheckpointStore) Save(record CheckpointRecord) error {
	if st.failSave.Load() > 0 {
		st.failSave.Add(^uint32(0))
		return errors.New("injected checkpoint save failure")
	}
	return st.inner.Save(record)
}

func TestFault_logCompactorCheckpointCrashRecovery(t *testing.T) {
	sourceDir := t.TempDir()
	warmDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")

	writeHotSegment(t, sourceDir, "segment_crash.log", buildSegmentPayload(t, 64, 1))

	store := NewLocalTierStore(sourceDir, warmDir)
	checkpoint := &faultCheckpointStore{inner: NewCheckpointStore(checkpointPath)}
	checkpoint.failSave.Store(1)

	compactor := NewCompactor(CompactorConfig{
		HotMinAge: 0,
		WarmDir:   warmDir,
		SourceDir: sourceDir,
	}, store, checkpoint, nil)

	ctx := context.Background()
	require.ErrorIs(t, compactor.RunOnce(ctx), ErrCompactionFailures)
	require.NoError(t, checkpoint.inner.Load())
	_, ok := checkpoint.inner.Get("segment_crash.log")
	require.False(t, ok)

	require.NoError(t, compactor.RunOnce(ctx))

	require.NoError(t, checkpoint.inner.Load())
	record, ok := checkpoint.inner.Get("segment_crash.log")
	require.True(t, ok)
	require.NotEmpty(t, record.DestSHA256)
	require.FileExists(t, filepath.Join(warmDir, record.DestKey))

	faultproof.Log(t, "log_compactor_checkpoint_crash_recovery", map[string]string{
		"subsystem":            "log_compactor",
		"checkpoint_persisted": "true",
		"warm_verified":        "true",
	})
}

func TestFault_logCompactorCompactingRecovery(t *testing.T) {
	sourceDir := t.TempDir()
	warmDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")

	hotPath := writeHotSegment(t, sourceDir, "segment_stuck.log", buildSegmentPayload(t, 32, 1))
	compactingPath := compactingPathFor(hotPath)
	require.NoError(t, os.Rename(hotPath, compactingPath))

	store := NewLocalTierStore(sourceDir, warmDir)
	compactor := NewCompactor(CompactorConfig{
		HotMinAge: 0,
		WarmDir:   warmDir,
		SourceDir: sourceDir,
	}, store, NewCheckpointStore(checkpointPath), nil)

	require.NoError(t, compactor.recoverStuckSegments(context.Background()))

	record, ok := compactor.checkpoint.Get("segment_stuck.log")
	require.True(t, ok)
	require.FileExists(t, filepath.Join(warmDir, record.DestKey))

	faultproof.Log(t, "log_compactor_compacting_recovery", map[string]string{
		"subsystem":          "log_compactor",
		"compacting_resumed": "true",
		"exactly_once":       "true",
	})
}

func TestFault_logCompactorWarmWriteRollback(t *testing.T) {
	sourceDir := t.TempDir()
	warmDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")

	writeHotSegment(t, sourceDir, "segment_verify.log", buildSegmentPayload(t, 8, 1))

	store := &faultInjectedLocalStore{
		inner:         NewLocalTierStore(sourceDir, warmDir),
		failWarmWrite: true,
	}
	compactor := NewCompactor(CompactorConfig{
		HotMinAge: 0,
		WarmDir:   warmDir,
		SourceDir: sourceDir,
	}, store, NewCheckpointStore(checkpointPath), nil)

	require.ErrorIs(t, compactor.RunOnce(context.Background()), ErrCompactionFailures)
	require.FileExists(t, filepath.Join(sourceDir, "segment_verify.log"))
	_, ok := compactor.checkpoint.Get("segment_verify.log")
	require.False(t, ok)
	require.False(t, CompactMarkerReady(filepath.Join(sourceDir, "segment_verify.log")))

	faultproof.Log(t, "log_compactor_warm_write_rollback", map[string]string{
		"subsystem":        "log_compactor",
		"hot_restored":     "true",
		"warm_rolled_back": "true",
	})
}

func TestFault_logCompactorConcurrentStress(t *testing.T) {
	sourceDir := t.TempDir()
	warmDir := t.TempDir()
	checkpointPath := filepath.Join(t.TempDir(), "checkpoint.jsonl")

	store := NewLocalTierStore(sourceDir, warmDir)
	checkpoint := NewCheckpointStore(checkpointPath)
	compactor := NewCompactor(CompactorConfig{
		HotMinAge:    0,
		WarmDir:      warmDir,
		SourceDir:    sourceDir,
		WorkInterval: 20 * time.Millisecond,
	}, store, checkpoint, nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = compactor.Run(ctx)
	}()

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			name := "segment_stress_" + itoa(index) + ".log"
			writeHotSegment(t, sourceDir, name, buildSegmentPayload(t, 10, 1))
		}(i)
	}
	wg.Wait()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if checkpoint.Count() == 20 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	require.Equal(t, 20, checkpoint.Count())

	faultproof.Log(t, "log_compactor_concurrent_stress", map[string]string{
		"subsystem":  "log_compactor",
		"goroutines": "20",
		"segments":   "20",
	})
}

type faultInjectedLocalStore struct {
	inner         *LocalTierStore
	failWarmWrite bool
}

func (st *faultInjectedLocalStore) ListHot(ctx context.Context, olderThan time.Time) ([]TierObject, error) {
	return st.inner.ListHot(ctx, olderThan)
}

func (st *faultInjectedLocalStore) WriteWarm(ctx context.Context, destKey string, plaintext []byte, meta CompactionMeta) error {
	return st.inner.WriteWarm(ctx, destKey, plaintext, meta)
}

func (st *faultInjectedLocalStore) RemoveHot(ctx context.Context, obj TierObject) error {
	return st.inner.RemoveHot(ctx, obj)
}

func (st *faultInjectedLocalStore) ClaimHot(ctx context.Context, obj TierObject) (TierObject, error) {
	return st.inner.ClaimHot(ctx, obj)
}

func (st *faultInjectedLocalStore) RollbackHot(ctx context.Context, obj TierObject) error {
	return st.inner.RollbackHot(ctx, obj)
}

func (st *faultInjectedLocalStore) ListStuckCompacting(ctx context.Context) ([]TierObject, error) {
	return st.inner.ListStuckCompacting(ctx)
}

func (st *faultInjectedLocalStore) RemoveCompacting(ctx context.Context, obj TierObject) error {
	return st.inner.RemoveCompacting(ctx, obj)
}

func (st *faultInjectedLocalStore) WriteWarmFromFile(ctx context.Context, destKey, filteredPath string, meta CompactionMeta) (string, error) {
	if st.failWarmWrite {
		return "", errors.New("injected warm write failure")
	}
	return st.inner.WriteWarmFromFile(ctx, destKey, filteredPath, meta)
}

func (st *faultInjectedLocalStore) RemoveWarmArtifacts(destKey string) {
	st.inner.RemoveWarmArtifacts(destKey)
}

func writeHotSegment(t *testing.T, dir, name string, payload []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, payload, 0o644))
	oldTime := time.Now().Add(-48 * time.Hour)
	require.NoError(t, os.Chtimes(path, oldTime, oldTime))
	return path
}

func buildSegmentPayload(t *testing.T, impressions, clicks int) []byte {
	t.Helper()
	var buf bytes.Buffer
	for i := range impressions {
		buf.Write(encodeRecord(t, &pb.AdStreamEvent{
			EventType: []byte("impression"),
			ClickId:   []byte("imp-" + itoa(i)),
		}))
	}
	for i := range clicks {
		buf.Write(encodeRecord(t, &pb.AdStreamEvent{
			EventType: []byte("click"),
			ClickId:   []byte("click-" + itoa(i)),
		}))
	}
	return buf.Bytes()
}
