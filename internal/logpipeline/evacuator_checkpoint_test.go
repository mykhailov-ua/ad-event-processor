package logpipeline

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckpointStore_saveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checkpoint")
	store := NewEvacuatorCheckpointStore(path)

	record := EvacuatorCheckpointRecord{
		FileName: "segment_20260101.log.zst",
		SHA256:   "abc123",
	}
	if err := store.Save(record); err != nil {
		t.Fatalf("save checkpoint: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load checkpoint: %v", err)
	}
	if loaded.FileName != record.FileName || loaded.SHA256 != record.SHA256 {
		t.Fatalf("checkpoint mismatch: got %+v want %+v", loaded, record)
	}
}

func TestCheckpointStore_missingFile(t *testing.T) {
	store := NewEvacuatorCheckpointStore(filepath.Join(t.TempDir(), "missing"))
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("load missing checkpoint: %v", err)
	}
	if loaded.FileName != "" || loaded.SHA256 != "" {
		t.Fatalf("expected empty checkpoint, got %+v", loaded)
	}
}

func TestCheckpointStore_corruptFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "checkpoint")
	if err := os.WriteFile(path, []byte("bad|line\n"), 0o644); err != nil {
		t.Fatalf("write corrupt checkpoint: %v", err)
	}

	store := NewEvacuatorCheckpointStore(path)
	_, err := store.Load()
	if !errors.Is(err, ErrEvacuatorCheckpointCorrupt) {
		t.Fatalf("expected ErrEvacuatorCheckpointCorrupt, got %v", err)
	}
}
