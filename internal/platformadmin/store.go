package platformadmin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const configRestartPendingKey = "platform_config_restart_pending"

type Store struct {
	pool *pgxpool.Pool
	host Host
}

func NewStore(pool *pgxpool.Pool, host Host) *Store {
	return &Store{pool: pool, host: host}
}

func (st *Store) poolOrNil() *pgxpool.Pool {
	if st == nil {
		return nil
	}
	return st.pool
}

func (st *Store) GetConfig(ctx context.Context) (platformconfig.Config, bool, error) {
	cfg, bootstrapped, err := st.loadConfig(ctx)
	if err != nil {
		return platformconfig.Config{}, false, err
	}
	return cfg, bootstrapped, nil
}

func (st *Store) GetRestartPending(ctx context.Context) ([]string, error) {
	if st.poolOrNil() == nil {
		return nil, nil
	}
	val, err := db.New(st.pool).GetSystemSetting(ctx, configRestartPendingKey)
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

func (st *Store) Bootstrap(ctx context.Context, req platformconfig.BootstrapRequest, installToken string) error {
	if st.poolOrNil() == nil || st.host == nil {
		return fmt.Errorf("service unavailable")
	}
	if err := st.host.VerifyInstallToken(installToken); err != nil {
		return err
	}
	_, bootstrapped, err := st.loadConfig(ctx)
	if err != nil {
		return err
	}
	if bootstrapped {
		return ErrConfigBootstrapped
	}
	cfg := platformconfig.MergeDefaults(req.Config)
	if err := cfg.Validate(); err != nil {
		return st.host.ErrValidation(err.Error())
	}
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		_, bootstrapped, err := st.loadConfigTx(ctx, q)
		if err != nil {
			return err
		}
		if bootstrapped {
			return ErrConfigBootstrapped
		}
		if err := st.saveConfigTx(ctx, q, cfg); err != nil {
			return err
		}
		if req.EulaVersion != "" {
			if err := st.host.SaveBootstrapEula(ctx, q, req.EulaVersion, req.AdminEmail); err != nil {
				return err
			}
		}
		st.host.AuditBootstrap(ctx, q, st.host.ActorUserID(ctx), cfg)
		return nil
	})
	if err != nil {
		return err
	}
	return st.host.SyncEdgeExpose(ctx, cfg)
}

func (st *Store) Update(ctx context.Context, patch platformconfig.Patch) (platformconfig.Config, []string, error) {
	if st.poolOrNil() == nil || st.host == nil {
		return platformconfig.Config{}, nil, fmt.Errorf("service unavailable")
	}
	before, bootstrapped, err := st.loadConfig(ctx)
	if err != nil {
		return platformconfig.Config{}, nil, err
	}
	if !bootstrapped {
		return platformconfig.Config{}, nil, ErrConfigNotBootstrapped
	}
	after, err := patch.Apply(before)
	if err != nil {
		return platformconfig.Config{}, nil, st.host.ErrValidation(err.Error())
	}
	restartRequired := platformconfig.RestartRequiredFields(before, after)
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := st.saveConfigTx(ctx, q, after); err != nil {
			return err
		}
		if err := st.saveRestartPendingTx(ctx, q, restartRequired); err != nil {
			return err
		}
		st.host.AuditUpdate(ctx, q, st.host.ActorUserID(ctx), before, after, restartRequired)
		return nil
	})
	if err != nil {
		return platformconfig.Config{}, nil, err
	}
	if syncErr := st.host.SyncEdgeExpose(ctx, after); syncErr != nil {
		return platformconfig.Config{}, nil, syncErr
	}
	return after, restartRequired, nil
}

func (st *Store) Apply(ctx context.Context, installRoot string) (string, error) {
	if st.poolOrNil() == nil || st.host == nil {
		return "", fmt.Errorf("service unavailable")
	}
	cfg, bootstrapped, err := st.loadConfig(ctx)
	if err != nil {
		return "", err
	}
	if !bootstrapped {
		return "", ErrConfigNotBootstrapped
	}
	root := platformconfig.FormatInstallRoot(installRoot)
	path := platformconfig.ComposeEnvPath(root)
	data := platformconfig.RenderComposeEnv(cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create install compose env dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write install compose env: %w", err)
	}
	err = pgx.BeginFunc(ctx, st.pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := st.saveRestartPendingTx(ctx, q, nil); err != nil {
			return err
		}
		st.host.AuditApply(ctx, q, st.host.ActorUserID(ctx), path)
		return nil
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (st *Store) loadConfig(ctx context.Context) (platformconfig.Config, bool, error) {
	if st.poolOrNil() == nil {
		return platformconfig.Default(), false, nil
	}
	return st.loadConfigTx(ctx, db.New(st.pool))
}

func (st *Store) loadConfigTx(ctx context.Context, q db.Querier) (platformconfig.Config, bool, error) {
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

func (st *Store) saveConfigTx(ctx context.Context, q db.Querier, cfg platformconfig.Config) error {
	raw, err := platformconfig.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal platform config: %w", err)
	}
	return q.SetSystemSetting(ctx, db.SetSystemSettingParams{
		Key:   platformconfig.SettingsKey,
		Value: raw,
	})
}

func (st *Store) saveRestartPendingTx(ctx context.Context, q db.Querier, pending []string) error {
	raw, err := coldpath.MarshalJSON(pending)
	if err != nil {
		return fmt.Errorf("marshal platform restart pending: %w", err)
	}
	return q.SetSystemSetting(ctx, db.SetSystemSettingParams{
		Key:   configRestartPendingKey,
		Value: string(raw),
	})
}
