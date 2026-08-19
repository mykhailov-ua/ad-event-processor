package trialregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Registry struct {
	path     string
	cooldown time.Duration
	mu       sync.Mutex
}

func New(path string, cooldown time.Duration) *Registry {
	if cooldown <= 0 {
		cooldown = defaultHWIDCooldownDays * 24 * time.Hour
	}
	return &Registry{
		path:     path,
		cooldown: cooldown,
	}
}

func NewFromConfig(cfg Config) *Registry {
	return New(cfg.RegistryPath, cfg.HWIDCooldown)
}

func (r *Registry) Path() string {
	return r.path
}

func (r *Registry) CheckPilotEligible(in CheckInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return err
	}
	return checkPilotEligible(time.Now().UTC(), r.cooldown, snap.Anchors, in)
}

func (r *Registry) RecordPilotIssue(in RecordInput) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	deploymentID := strings.TrimSpace(in.DeploymentID)
	if deploymentID == "" {
		return fmt.Errorf("deployment_id is required")
	}

	issuedAt := now
	validUntil := in.ValidUntil.UTC()
	if validUntil.IsZero() {
		validUntil = now
	}

	upsertAnchor := func(anchorType AnchorType, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		rec := AnchorRecord{
			AnchorType:   anchorType,
			AnchorValue:  value,
			DeploymentID: deploymentID,
			LicenseKey:   strings.TrimSpace(in.LicenseKey),
			IssuedAt:     issuedAt,
			ValidUntil:   validUntil,
			Status:       StatusActive,
		}
		snap.Anchors = upsertAnchorRecord(snap.Anchors, rec)
	}

	upsertAnchor(AnchorDeploymentID, deploymentID)
	upsertAnchor(AnchorTelegram, in.TelegramID)
	upsertAnchor(AnchorHWID, in.HWID)
	upsertAnchor(AnchorUSDTTx, in.USDTTx)

	if in.Force {
		snap.Overrides = append(snap.Overrides, OverrideRecord{
			DeploymentID: deploymentID,
			Reason:       strings.TrimSpace(in.ForceReason),
			Operator:     strings.TrimSpace(in.Operator),
			At:           now,
		})
	}

	return r.saveLocked(snap)
}

func (r *Registry) RecordHWID(deploymentID, hwid string) error {
	deploymentID = strings.TrimSpace(deploymentID)
	hwid = strings.TrimSpace(hwid)
	if deploymentID == "" || hwid == "" {
		return fmt.Errorf("deployment_id and hwid are required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	rec := AnchorRecord{
		AnchorType:   AnchorHWID,
		AnchorValue:  hwid,
		DeploymentID: deploymentID,
		IssuedAt:     now,
		ValidUntil:   now,
		Status:       StatusActive,
	}
	snap.Anchors = upsertAnchorRecord(snap.Anchors, rec)
	return r.saveLocked(snap)
}

func (r *Registry) MarkConverted(deploymentID string) error {
	return r.setDeploymentStatus(deploymentID, StatusConverted)
}

func (r *Registry) MarkExpired(deploymentID string) error {
	return r.setDeploymentStatus(deploymentID, StatusExpired)
}

func (r *Registry) setDeploymentStatus(deploymentID string, status Status) error {
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return fmt.Errorf("deployment_id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return err
	}

	changed := false
	for i := range snap.Anchors {
		if snap.Anchors[i].DeploymentID != deploymentID {
			continue
		}
		snap.Anchors[i].Status = status
		changed = true
	}
	if !changed {
		return fmt.Errorf("deployment_id not found in trial registry: %s", deploymentID)
	}
	return r.saveLocked(snap)
}

func upsertAnchorRecord(anchors []AnchorRecord, rec AnchorRecord) []AnchorRecord {
	for i := range anchors {
		if anchors[i].AnchorType == rec.AnchorType && anchors[i].AnchorValue == rec.AnchorValue {
			anchors[i] = rec
			return anchors
		}
	}
	return append(anchors, rec)
}

func (r *Registry) loadLocked() (*fileSnapshot, error) {
	data, err := os.ReadFile(r.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &fileSnapshot{Version: 1}, nil
		}
		return nil, fmt.Errorf("read trial registry: %w", err)
	}
	if len(data) == 0 {
		return &fileSnapshot{Version: 1}, nil
	}
	var snap fileSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, fmt.Errorf("parse trial registry: %w", err)
	}
	if snap.Version == 0 {
		snap.Version = 1
	}
	return &snap, nil
}

func (r *Registry) saveLocked(snap *fileSnapshot) error {
	if snap.Version == 0 {
		snap.Version = 1
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("encode trial registry: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create trial registry dir: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".trial_registry-*.tmp")
	if err != nil {
		return fmt.Errorf("create trial registry temp: %w", err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write trial registry temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close trial registry temp: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod trial registry temp: %w", err)
	}
	if err := os.Rename(tmpName, r.path); err != nil {
		return fmt.Errorf("rename trial registry: %w", err)
	}
	return nil
}
