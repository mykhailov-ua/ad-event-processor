// Package legal embeds operator-facing license and EULA text.
package legal

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

const (
	SettingsKey = "eula_acceptance"
	Version     = "2026-01"
)

//go:embed EULA.txt
var Text string

type Acceptance struct {
	Version    string    `json:"version"`
	AcceptedAt time.Time `json:"accepted_at"`
	AcceptedBy string    `json:"accepted_by"`
}

func ParseAcceptance(raw string) (Acceptance, error) {
	if raw == "" {
		return Acceptance{}, fmt.Errorf("empty eula acceptance")
	}
	var a Acceptance
	if err := json.Unmarshal([]byte(raw), &a); err != nil {
		return Acceptance{}, fmt.Errorf("parse eula acceptance: %w", err)
	}
	if a.Version == "" {
		return Acceptance{}, fmt.Errorf("eula acceptance missing version")
	}
	return a, nil
}

func MarshalAcceptance(a Acceptance) (string, error) {
	data, err := json.Marshal(a)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func IsCurrent(a Acceptance) bool {
	return a.Version == Version
}
