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

func (e *Evacuator) Run(ctx context.Context) error {
	if err := os.MkdirAll(e.cfg.LogDir, 0o755); err != nil {
		return err
	}
	if err := e.watcher.Add(e.cfg.LogDir); err != nil {
		return err
	}
	defer func() { _ = e.watcher.Close() }()

	if err := e.recoverStuckSegments(ctx); err != nil {
		slog.Warn("recover stuck segments", "error", err)
	}
	if err := e.scanReadySegments(ctx); err != nil {
		slog.Warn("initial ready scan failed", "error", err)
	}

	scanTicker := time.NewTicker(e.cfg.ScanInterval)
	defer scanTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-e.watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) || event.Has(fsnotify.Write) {
				if strings.HasSuffix(event.Name, readySuffix) {
					if err := e.processReadyFile(ctx, event.Name); err != nil {
						slog.Warn("process ready segment failed", "path", event.Name, "error", err)
					}
				}
			}
		case err, ok := <-e.watcher.Errors:
			if !ok {
				return nil
			}
			slog.Warn("fsnotify error", "error", err)
		case <-scanTicker.C:
			if err := e.scanReadySegments(ctx); err != nil {
				slog.Warn("ready segment scan failed", "error", err)
			}
		}
	}
}

func (e *Evacuator) recoverStuckSegments(ctx context.Context) error {
	entries, err := os.ReadDir(e.cfg.LogDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, evacuatingSuffix) {
			path := filepath.Join(e.cfg.LogDir, name)
			if err := e.processEvacuatingFile(ctx, path); err != nil {
				slog.Warn("recover evacuating segment failed", "path", path, "error", err)
			}
		}
	}

	return nil
}

func (e *Evacuator) scanReadySegments(ctx context.Context) error {
	entries, err := os.ReadDir(e.cfg.LogDir)
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
			readyPaths = append(readyPaths, filepath.Join(e.cfg.LogDir, name))
		}
	}

	sort.Strings(readyPaths)
	for _, path := range readyPaths {
		if err := e.processReadyFile(ctx, path); err != nil {
			slog.Warn("process ready segment failed", "path", path, "error", err)
		}
	}

	return nil
}

func (e *Evacuator) processReadyFile(ctx context.Context, readyPath string) error {
	if e.cfg.RequireCompactorMarker && !CompactMarkerReady(readyPath) {
		return nil
	}

	evacPath, err := e.claimReadyFile(readyPath)
	if err != nil {
		return err
	}
	if evacPath == "" {
		return nil
	}

	return e.uploadSegment(ctx, evacPath)
}

func (e *Evacuator) processEvacuatingFile(ctx context.Context, evacPath string) error {
	e.mu.Lock()
	if _, exists := e.inflight[evacPath]; exists {
		e.mu.Unlock()
		return ErrEvacuatingInUse
	}
	e.inflight[evacPath] = struct{}{}
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.inflight, evacPath)
		e.mu.Unlock()
	}()

	return e.uploadSegment(ctx, evacPath)
}

func (e *Evacuator) claimReadyFile(readyPath string) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.inflight[readyPath]; exists {
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

	e.inflight[evacuatingPath] = struct{}{}
	return evacuatingPath, nil
}

func (e *Evacuator) uploadSegment(ctx context.Context, evacPath string) error {
	defer func() {
		e.mu.Lock()
		delete(e.inflight, evacPath)
		e.mu.Unlock()
	}()

	digests, err := computeFileDigests(evacPath)
	if err != nil {
		return e.rollback(evacPath, err)
	}

	objectKey := segmentObjectKey(evacPath)
	head, err := e.store.HeadObject(ctx, objectKey)
	if err != nil {
		return e.rollback(evacPath, err)
	}
	if head.Exists && head.SHA256 == digests.SHA256 {
		if err := e.finalize(evacPath, objectKey, digests); err != nil {
			return e.rollback(evacPath, err)
		}
		return nil
	}

	if err := e.store.PutObject(ctx, objectKey, evacPath, digests); err != nil {
		return e.rollback(evacPath, err)
	}

	verifyHead, err := e.store.HeadObject(ctx, objectKey)
	if err != nil {
		return e.rollback(evacPath, err)
	}
	if !verifyHead.Exists || verifyHead.SHA256 != digests.SHA256 {
		return e.rollback(evacPath, ErrDigestMismatch)
	}

	return e.finalize(evacPath, objectKey, digests)
}

func (e *Evacuator) finalize(evacPath, objectKey string, digests fileDigests) error {
	record := EvacuatorCheckpointRecord{
		FileName: filepath.Base(objectKey),
		SHA256:   digests.SHA256,
	}
	if err := e.checkpoint.Save(record); err != nil {
		return err
	}
	return os.Remove(evacPath)
}

func (e *Evacuator) rollback(evacPath string, cause error) error {
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
