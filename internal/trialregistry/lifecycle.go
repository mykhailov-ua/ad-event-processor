package trialregistry

import (
	"strings"
	"time"
)

// ExpireStale marks active anchors expired when valid_until is before at.
// Returns the number of anchor rows updated.
func (r *Registry) ExpireStale(at time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return 0, err
	}

	at = at.UTC()
	changed := 0
	for i := range snap.Anchors {
		if snap.Anchors[i].Status != StatusActive {
			continue
		}
		until := snap.Anchors[i].ValidUntil.UTC()
		if until.IsZero() || !until.Before(at) {
			continue
		}
		snap.Anchors[i].Status = StatusExpired
		changed++
	}
	if changed == 0 {
		return 0, nil
	}
	if err := r.saveLocked(snap); err != nil {
		return 0, err
	}
	return changed, nil
}

// DeploymentHasStatus reports whether any anchor exists for deployment_id.
func (r *Registry) DeploymentHasStatus(deploymentID string, status Status) (bool, error) {
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return false, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	snap, err := r.loadLocked()
	if err != nil {
		return false, err
	}
	for _, rec := range snap.Anchors {
		if rec.DeploymentID == deploymentID && rec.Status == status {
			return true, nil
		}
	}
	return false, nil
}
