package licensingadmin

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/legal"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrEulaVersionMismatch = errors.New("eula version mismatch")

func GetEulaStatus(ctx context.Context, host EulaHost) (legal.Acceptance, bool, error) {
	if host == nil || host.EulaPool() == nil {
		return legal.Acceptance{}, false, nil
	}
	raw, err := db.New(host.EulaPool()).GetSystemSetting(ctx, legal.SettingsKey)
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

func AcceptEula(ctx context.Context, host EulaHost, version, acceptedBy string) error {
	if host == nil || host.EulaPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = legal.Version
	}
	if version != legal.Version {
		return ErrEulaVersionMismatch
	}
	return pgx.BeginFunc(ctx, host.EulaPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if err := SaveEulaAcceptanceTx(ctx, host, q, version, acceptedBy); err != nil {
			return err
		}
		host.EulaAuditAccept(ctx, q, host.EulaActorID(ctx), version, acceptedBy)
		return nil
	})
}

func SaveEulaAcceptanceTx(ctx context.Context, host EulaHost, q db.Querier, version, acceptedBy string) error {
	version = strings.TrimSpace(version)
	if version != legal.Version {
		return ErrEulaVersionMismatch
	}
	acceptedBy = strings.TrimSpace(acceptedBy)
	if acceptedBy == "" {
		if host != nil {
			acceptedBy = host.EulaActorID(ctx).String()
		}
		if acceptedBy == "" || acceptedBy == uuid.Nil.String() {
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
