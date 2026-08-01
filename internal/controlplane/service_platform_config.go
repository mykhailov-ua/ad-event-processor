package controlplane

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"espx/internal/domain/db"
	"espx/pkg/coldpath"
	"espx/pkg/platformconfig"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const platformConfigRestartPendingKey = "platform_config_restart_pending"

var (
	ErrPlatformConfigBootstrapped    = errors.New("platform config already bootstrapped")
	ErrPlatformConfigNotBootstrapped = errors.New("platform config not bootstrapped")
	ErrInstallTokenInvalid           = errors.New("invalid install token")
)

func (s *Service) GetPlatformConfig(ctx context.Context) (platformconfig.Config, bool, error) {
	cfg, bootstrapped, err := s.loadPlatformConfig(ctx)
	if err != nil {
		return platformconfig.Config{}, false, err
	}
	return cfg, bootstrapped, nil
}

func (s *Service) GetPlatformRestartPending(ctx context.Context) ([]string, error) {
	if s == nil || s.GetPool() == nil {
		return nil, nil
	}
	val, err := db.New(s.GetPool()).GetSystemSetting(ctx, platformConfigRestartPendingKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if val == "" || val == "[]" {
		return nil, nil
	}
	var pending []string
	if err := coldpath.UnmarshalJSON([]byte(val), &pending); err != nil {
		return nil, fmt.Errorf("parse platform restart pending: %w", err)
	}
	return pending, nil
}

func (s *Service) BootstrapPlatformConfig(ctx context.Context, req platformconfig.BootstrapRequest, installToken string) error {
	if err := s.verifyInstallToken(installToken); err != nil {
		return err
	}
	_, bootstrapped, err := s.loadPlatformConfig(ctx)
	if err != nil {
		return err
	}
	if bootstrapped {
		return ErrPlatformConfigBootstrapped
	}
	cfg := platformconfig.MergeDefaults(req.Config)
	if err := cfg.Validate(); err != nil {
		return errValidation(err.Error())
	}
	return pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		_, bootstrapped, err := s.loadPlatformConfigTx(ctx, q)
		if err != nil {
			return err
		}
		if bootstrapped {
			return ErrPlatformConfigBootstrapped
		}
		if err := s.savePlatformConfigTx(ctx, q, cfg); err != nil {
			return err
		}
		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PLATFORM_CONFIG_BOOTSTRAP", "system", nil, platformconfig.RedactConfig(cfg), nil)
		return nil
	})
}

func (s *Service) UpdatePlatformConfig(ctx context.Context, patch platformconfig.Patch) (platformconfig.Config, []string, error) {
	before, bootstrapped, err := s.loadPlatformConfig(ctx)
	if err != nil {
		return platformconfig.Config{}, nil, err
	}
	if !bootstrapped {
		return platformconfig.Config{}, nil, ErrPlatformConfigNotBootstrapped
	}
	after, err := patch.Apply(before)
	if err != nil {
		return platformconfig.Config{}, nil, errValidation(err.Error())
	}
	restartRequired := platformconfig.RestartRequiredFields(before, after)
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := s.savePlatformConfigTx(ctx, q, after); err != nil {
			return err
		}
		if err := s.savePlatformRestartPendingTx(ctx, q, restartRequired); err != nil {
			return err
		}
		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PLATFORM_CONFIG_UPDATE", "system", nil, map[string]any{
			"before":           platformconfig.RedactConfig(before),
			"after":            platformconfig.RedactConfig(after),
			"restart_required": restartRequired,
		}, nil)
		return nil
	})
	if err != nil {
		return platformconfig.Config{}, nil, err
	}
	return after, restartRequired, nil
}

func (s *Service) ApplyPlatformConfig(ctx context.Context, installRoot string) (string, error) {
	cfg, bootstrapped, err := s.loadPlatformConfig(ctx)
	if err != nil {
		return "", err
	}
	if !bootstrapped {
		return "", ErrPlatformConfigNotBootstrapped
	}
	root := platformconfig.FormatInstallRoot(installRoot)
	path := platformconfig.ComposeEnvPath(root)
	data := platformconfig.RenderComposeEnv(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", fmt.Errorf("create install compose env dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write install compose env: %w", err)
	}
	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := s.savePlatformRestartPendingTx(ctx, q, nil); err != nil {
			return err
		}
		var uid uuid.UUID
		if u, ok := GetUser(ctx); ok {
			uid = u.UserID
		}
		s.AuditLog(ctx, q, uid, "PLATFORM_CONFIG_APPLY", "system", nil, map[string]string{
			"written_path": path,
		}, nil)
		return nil
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (s *Service) verifyInstallToken(token string) error {
	if s == nil || s.cfg == nil || len(s.cfg.InstallBootstrapToken) == 0 {
		return ErrInstallTokenInvalid
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.InstallBootstrapToken)) != 1 {
		return ErrInstallTokenInvalid
	}
	return nil
}

func (s *Service) loadPlatformConfig(ctx context.Context) (platformconfig.Config, bool, error) {
	if s == nil || s.GetPool() == nil {
		return platformconfig.Default(), false, nil
	}
	return s.loadPlatformConfigTx(ctx, db.New(s.GetPool()))
}

func (s *Service) loadPlatformConfigTx(ctx context.Context, q db.Querier) (platformconfig.Config, bool, error) {
	val, err := q.GetSystemSetting(ctx, platformconfig.SettingsKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return platformconfig.Default(), false, nil
	}
	if err != nil {
		return platformconfig.Config{}, false, err
	}
	cfg, err := platformconfig.Parse(val)
	if err != nil {
		return platformconfig.Config{}, false, err
	}
	return cfg, true, nil
}

func (s *Service) savePlatformConfigTx(ctx context.Context, q db.Querier, cfg platformconfig.Config) error {
	raw, err := platformconfig.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal platform config: %w", err)
	}
	return q.SetSystemSetting(ctx, db.SetSystemSettingParams{
		Key:   platformconfig.SettingsKey,
		Value: raw,
	})
}

func (s *Service) savePlatformRestartPendingTx(ctx context.Context, q db.Querier, pending []string) error {
	raw, err := coldpath.MarshalJSON(pending)
	if err != nil {
		return fmt.Errorf("marshal platform restart pending: %w", err)
	}
	return q.SetSystemSetting(ctx, db.SetSystemSettingParams{
		Key:   platformConfigRestartPendingKey,
		Value: string(raw),
	})
}
