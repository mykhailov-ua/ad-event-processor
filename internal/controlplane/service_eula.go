package controlplane

import (
	"context"
	"errors"
	"fmt"
	"time"

	"espx/internal/domain/db"
	"espx/pkg/legal"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrEulaVersionMismatch = errors.New("eula version mismatch")

func (s *Service) GetEulaStatus(ctx context.Context) (legal.Acceptance, bool, error) {
	if s == nil || s.GetPool() == nil {
		return legal.Acceptance{}, false, nil
	}
	raw, err := db.New(s.GetPool()).GetSystemSetting(ctx, legal.SettingsKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return legal.Acceptance{}, false, nil
	}
	if err != nil {
		return legal.Acceptance{}, false, err
	}
	acc, err := legal.ParseAcceptance(raw)
	if err != nil {
		return legal.Acceptance{}, false, err
	}
	return acc, legal.IsCurrent(acc), nil
}

func (s *Service) AcceptEula(ctx context.Context, version, acceptedBy string) error {
	if s == nil || s.GetPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	version = trimSpace(version)
	if version == "" {
		version = legal.Version
	}
	if version != legal.Version {
		return ErrEulaVersionMismatch
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := s.saveEulaAcceptanceTx(ctx, q, version, acceptedBy); err != nil {
			return err
		}
		var adminID uuid.UUID
		if u, ok := GetUser(ctx); ok {
			adminID = u.UserID
		}
		s.AuditLog(ctx, q, adminID, "EULA_ACCEPT", "system", nil, map[string]string{
			"version": version,
			"by":      acceptedBy,
		}, nil)
		return nil
	})
}

func (s *Service) saveEulaAcceptanceTx(ctx context.Context, q db.Querier, version, acceptedBy string) error {
	version = trimSpace(version)
	if version != legal.Version {
		return ErrEulaVersionMismatch
	}
	acceptedBy = trimSpace(acceptedBy)
	if acceptedBy == "" {
		if u, ok := GetUser(ctx); ok {
			acceptedBy = u.UserID.String()
		} else {
			acceptedBy = "install"
		}
	}
	acc := legal.Acceptance{
		Version:    version,
		AcceptedAt: time.Now().UTC(),
		AcceptedBy: acceptedBy,
	}
	raw, err := legal.MarshalAcceptance(acc)
	if err != nil {
		return err
	}
	return q.SetSystemSetting(ctx, db.SetSystemSettingParams{
		Key:   legal.SettingsKey,
		Value: raw,
	})
}

func trimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t' || s[j-1] == '\n' || s[j-1] == '\r') {
		j--
	}
	return s[i:j]
}
