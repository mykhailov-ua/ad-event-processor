package logpipeline

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type EvacuatorConfig struct {
	LogDir                 string
	CheckpointPath         string
	ScanInterval           time.Duration
	MultipartThreshold     int64
	RequireCompactorMarker bool
}

type Evacuator struct {
	cfg        EvacuatorConfig
	store      ObjectStore
	checkpoint *EvacuatorCheckpointStore
	watcher    *fsnotify.Watcher
	mu         sync.Mutex
	inflight   map[string]struct{}
}

func NewEvacuator(cfg EvacuatorConfig, store ObjectStore) (*Evacuator, error) {
	if cfg.LogDir == "" {
		cfg.LogDir = "/var/log/ad-event-processor"
	}
	if cfg.CheckpointPath == "" {
		cfg.CheckpointPath = "/var/lib/ad-event-processor/log-evacuator.checkpoint"
	}
	if cfg.ScanInterval <= 0 {
		cfg.ScanInterval = 5 * time.Second
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Evacuator{
		cfg:        cfg,
		store:      store,
		checkpoint: NewEvacuatorCheckpointStore(cfg.CheckpointPath),
		watcher:    watcher,
		inflight:   make(map[string]struct{}),
	}, nil
}

func (evac *Evacuator) Run(ctx context.Context) error {
	if err := os.MkdirAll(evac.cfg.LogDir, 0o755); err != nil {
		return err
	}
	if err := evac.watcher.Add(evac.cfg.LogDir); err != nil {
		return err
	}
	defer evac.watcher.Close()

	if err := evac.recoverStuckSegments(ctx); err != nil {
		slog.Warn("recover stuck segments", "error", err)
	}
	if err := evac.scanReadySegments(ctx); err != nil {
		slog.Warn("initial ready scan failed", "error", err)
	}

	scanTicker := time.NewTicker(evac.cfg.ScanInterval)
	defer scanTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-evac.watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Write) {
				if strings.HasSuffix(event.Name, readySuffix) {
					if err := evac.processReadyFile(ctx, event.Name); err != nil {
						slog.Warn("process ready segment failed", "path", event.Name, "error", err)
					}
				}
			}
		case err, ok := <-evac.watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn("fsnotify error", "error", err)
		case <-scanTicker.C:
			if err := evac.scanReadySegments(ctx); err != nil {
				slog.Warn("ready segment scan failed", "error", err)
			}
		}
	}
}

func (evac *Evacuator) recoverStuckSegments(ctx context.Context) error {
	entries, err := os.ReadDir(evac.cfg.LogDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, evacuatingSuffix) {
			path := filepath.Join(evac.cfg.LogDir, name)
			if err := evac.processEvacuatingFile(ctx, path); err != nil {
				slog.Warn("recover evacuating segment failed", "path", path, "error", err)
			}
		}
	}

	return nil
}

func (evac *Evacuator) scanReadySegments(ctx context.Context) error {
	entries, err := os.ReadDir(evac.cfg.LogDir)
	if err != nil {
		return err
	}

	var readyPaths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, readySuffix) {
			readyPaths = append(readyPaths, filepath.Join(evac.cfg.LogDir, name))
		}
	}

	sort.Strings(readyPaths)
	for _, path := range readyPaths {
		if err := evac.processReadyFile(ctx, path); err != nil {
			slog.Warn("process ready segment failed", "path", path, "error", err)
		}
	}

	return nil
}

func (evac *Evacuator) processReadyFile(ctx context.Context, readyPath string) error {
	if evac.cfg.RequireCompactorMarker && !CompactMarkerReady(readyPath) {
		return nil
	}

	evacPath, err := evac.claimReadyFile(readyPath)
	if err != nil {
		return err
	}
	if evacPath == "" {
		return nil
	}

	return evac.uploadSegment(ctx, evacPath)
}

func (evac *Evacuator) processEvacuatingFile(ctx context.Context, evacPath string) error {
	evac.mu.Lock()
	if _, exists := evac.inflight[evacPath]; exists {
		evac.mu.Unlock()
		return ErrEvacuatingInUse
	}
	evac.inflight[evacPath] = struct{}{}
	evac.mu.Unlock()

	defer func() {
		evac.mu.Lock()
		delete(evac.inflight, evacPath)
		evac.mu.Unlock()
	}()

	return evac.uploadSegment(ctx, evacPath)
}

func (evac *Evacuator) claimReadyFile(readyPath string) (string, error) {
	evac.mu.Lock()
	defer evac.mu.Unlock()

	if _, exists := evac.inflight[readyPath]; exists {
		return "", nil
	}

	if !strings.HasSuffix(readyPath, readySuffix) {
		return "", ErrNotReadySegment
	}

	evacuatingPath := strings.TrimSuffix(readyPath, readySuffix) + evacuatingSuffix
	if err := os.Rename(readyPath, evacuatingPath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}

	evac.inflight[evacuatingPath] = struct{}{}
	return evacuatingPath, nil
}

func (evac *Evacuator) uploadSegment(ctx context.Context, evacPath string) error {
	defer func() {
		evac.mu.Lock()
		delete(evac.inflight, evacPath)
		evac.mu.Unlock()
	}()

	digests, err := computeFileDigests(evacPath)
	if err != nil {
		return evac.rollback(evacPath, err)
	}

	objectKey := segmentObjectKey(evacPath)
	head, err := evac.store.HeadObject(ctx, objectKey)
	if err != nil {
		return evac.rollback(evacPath, err)
	}
	if head.Exists && head.SHA256 == digests.SHA256 {
		if err := evac.finalize(evacPath, objectKey, digests); err != nil {
			return evac.rollback(evacPath, err)
		}
		return nil
	}

	if err := evac.store.PutObject(ctx, objectKey, evacPath, digests); err != nil {
		return evac.rollback(evacPath, err)
	}

	verifyHead, err := evac.store.HeadObject(ctx, objectKey)
	if err != nil {
		return evac.rollback(evacPath, err)
	}
	if !verifyHead.Exists || verifyHead.SHA256 != digests.SHA256 {
		return evac.rollback(evacPath, ErrDigestMismatch)
	}

	return evac.finalize(evacPath, objectKey, digests)
}

func (evac *Evacuator) finalize(evacPath, objectKey string, digests fileDigests) error {
	record := EvacuatorCheckpointRecord{
		FileName: filepath.Base(objectKey),
		SHA256:   digests.SHA256,
	}
	if err := evac.checkpoint.Save(record); err != nil {
		return err
	}
	return os.Remove(evacPath)
}

func (evac *Evacuator) rollback(evacPath string, cause error) error {
	readyPath := strings.TrimSuffix(evacPath, evacuatingSuffix) + readySuffix
	if renameErr := os.Rename(evacPath, readyPath); renameErr != nil && !os.IsNotExist(renameErr) {
		slog.Error("rollback evacuating segment failed", "path", evacPath, "error", renameErr)
	}
	return cause
}

func segmentObjectKey(evacPath string) string {
	base := filepath.Base(evacPath)
	return strings.TrimSuffix(base, ".evacuating")
}
