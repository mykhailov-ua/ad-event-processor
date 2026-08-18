package trialregistry

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r *Registry) EnqueuePending(in EnqueuePendingInput) (PendingRequest, error) {
	telegramID := normalizeTelegramID(in.TelegramID)
	if telegramID == "" {
		return PendingRequest{}, fmt.Errorf("telegram_id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return PendingRequest{}, err
	}

	if err := telegramAnchorBlocksPilot(time.Now().UTC(), snap.Anchors, telegramID); err != nil {
		return PendingRequest{}, err
	}

	for i := range snap.Pending {
		if snap.Pending[i].Status != PendingStatusOpen {
			continue
		}
		if snap.Pending[i].TelegramID == telegramID {
			return snap.Pending[i], nil
		}
	}

	now := time.Now().UTC()
	req := PendingRequest{
		ID:               uuid.NewString(),
		TelegramID:       telegramID,
		TelegramUsername: strings.TrimSpace(in.TelegramUsername),
		RequestedAt:      now,
		Status:           PendingStatusOpen,
		Notes:            strings.TrimSpace(in.Notes),
	}
	snap.Pending = append(snap.Pending, req)
	if err := r.saveLocked(snap); err != nil {
		return PendingRequest{}, err
	}
	return req, nil
}

func (r *Registry) ListPending() ([]PendingRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return nil, err
	}
	out := make([]PendingRequest, 0, len(snap.Pending))
	for _, req := range snap.Pending {
		if req.Status == PendingStatusOpen {
			out = append(out, req)
		}
	}
	return out, nil
}

func (r *Registry) GetPending(id string) (PendingRequest, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return PendingRequest{}, fmt.Errorf("pending id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return PendingRequest{}, err
	}
	for _, req := range snap.Pending {
		if req.ID == id {
			return req, nil
		}
	}
	return PendingRequest{}, ErrPendingNotFound
}

// PreparePendingIssue marks a pending row approved and returns deployment_id for JWT issue.
func (r *Registry) PreparePendingIssue(id, deploymentID string) (PendingRequest, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return PendingRequest{}, fmt.Errorf("pending id is required")
	}
	deploymentID = strings.TrimSpace(deploymentID)

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return PendingRequest{}, err
	}

	idx := -1
	for i := range snap.Pending {
		if snap.Pending[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return PendingRequest{}, ErrPendingNotFound
	}
	req := snap.Pending[idx]
	if req.Status != PendingStatusOpen {
		return PendingRequest{}, ErrPendingNotOpen
	}

	if deploymentID == "" {
		deploymentID = strings.TrimSpace(req.DeploymentID)
	}
	if deploymentID == "" {
		deploymentID = uuid.NewString()
	}

	req.DeploymentID = deploymentID
	req.Status = PendingStatusApproved
	snap.Pending[idx] = req
	if err := r.saveLocked(snap); err != nil {
		return PendingRequest{}, err
	}
	return req, nil
}

func (r *Registry) RejectPending(id, reason string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("pending id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return err
	}

	idx := -1
	for i := range snap.Pending {
		if snap.Pending[i].ID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return ErrPendingNotFound
	}
	if snap.Pending[idx].Status != PendingStatusOpen {
		return ErrPendingNotOpen
	}

	snap.Pending[idx].Status = PendingStatusRejected
	if note := strings.TrimSpace(reason); note != "" {
		snap.Pending[idx].Notes = note
	}
	return r.saveLocked(snap)
}

func telegramAnchorBlocksPilot(now time.Time, anchors []AnchorRecord, telegramID string) error {
	telegramID = normalizeTelegramID(telegramID)
	if telegramID == "" {
		return nil
	}
	for _, rec := range anchors {
		if rec.AnchorType != AnchorTelegram || rec.AnchorValue != telegramID {
			continue
		}
		switch rec.Status {
		case StatusActive, StatusExpired, StatusConverted:
			return ErrTrialTelegramUsed
		}
	}
	return nil
}

func normalizeTelegramID(raw string) string {
	return strings.TrimSpace(raw)
}
