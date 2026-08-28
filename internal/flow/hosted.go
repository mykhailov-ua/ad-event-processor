package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type landerPublishedPayload struct {
	LanderID string `json:"lander_id"`
}

func UploadHostedLanderZip(ctx context.Context, host HostedLanderHost, landerID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (LanderDTO, error) {
	pool := host.HostedLanderPool()
	if pool == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	st := host.HostedLanderStore()
	if st == nil {
		return LanderDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	if landerID == uuid.Nil {
		return LanderDTO{}, fmt.Errorf("lander id required")
	}
	maxBytes := host.LanderMaxZipBytes()
	if maxBytes <= 0 {
		maxBytes = landerhost.DefaultMaxZipBytes
	}
	if zipSize <= 0 || zipSize > maxBytes {
		return LanderDTO{}, fmt.Errorf("zip size must be between 1 and %d bytes", maxBytes)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return LanderDTO{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists int
	err = tx.QueryRow(ctx, `SELECT 1 FROM landers WHERE id = $1`, landerID).Scan(&exists)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return LanderDTO{}, fmt.Errorf("lander not found")
		}
		return LanderDTO{}, err
	}

	var nextVersion int
	err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM lander_assets WHERE lander_id = $1`, landerID).Scan(&nextVersion)
	if err != nil {
		return LanderDTO{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LanderDTO{}, err
	}

	entryCount, err := st.ExtractZip(landerID, nextVersion, zipReader, zipSize)
	if err != nil {
		return LanderDTO{}, err
	}

	tx2, err := pool.Begin(ctx)
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return LanderDTO{}, err
	}
	defer func() { _ = tx2.Rollback(ctx) }()

	var assetID uuid.UUID
	err = tx2.QueryRow(ctx, `
		INSERT INTO lander_assets (lander_id, version, entry_count)
		VALUES ($1, $2, $3)
		RETURNING id`, landerID, nextVersion, entryCount).Scan(&assetID)
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return LanderDTO{}, err
	}

	var dto LanderDTO
	err = tx2.QueryRow(ctx, `
		UPDATE landers
		SET hosted_asset_id = $2, url = ''
		WHERE id = $1
		RETURNING id, name, COALESCE(url, ''), hosted_asset_id, created_at`,
		landerID, assetID).Scan(&dto.ID, &dto.Name, &dto.URL, &dto.HostedAssetID, &dto.CreatedAt)
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return LanderDTO{}, err
	}

	payload, err := json.Marshal(landerPublishedPayload{LanderID: landerID.String()})
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return LanderDTO{}, err
	}
	if _, err := tx2.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload, status)
		VALUES ('LANDER_PUBLISHED', $1, 'PENDING')`, payload); err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return LanderDTO{}, err
	}

	if err := tx2.Commit(ctx); err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return LanderDTO{}, err
	}
	if err := st.PublishVersion(landerID, nextVersion); err != nil {
		return LanderDTO{}, err
	}

	dto.HostedURL = landerhost.PublicURL(host.LanderPublicBase(ctx), landerID)
	return dto, nil
}

func ServeHostedLanderFile(ctx context.Context, host HostedLanderHost, landerID uuid.UUID, relPath string) (io.ReadCloser, string, error) {
	st := host.HostedLanderStore()
	if st == nil {
		return nil, "", fmt.Errorf("hosted lander store is not configured")
	}
	if landerID == uuid.Nil {
		return nil, "", fmt.Errorf("lander id required")
	}
	pool := host.HostedLanderPool()
	if pool != nil {
		var assetID *uuid.UUID
		err := pool.QueryRow(ctx, `SELECT hosted_asset_id FROM landers WHERE id = $1`, landerID).Scan(&assetID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, "", fmt.Errorf("lander not found")
			}
			return nil, "", err
		}
		if assetID == nil || *assetID == uuid.Nil {
			return nil, "", fmt.Errorf("lander has no hosted asset")
		}
	}
	rc, info, err := st.OpenLiveFile(landerID, relPath)
	if err != nil {
		return nil, "", err
	}
	ctype := contentTypeForPath(relPath, info.Name())
	return rc, ctype, nil
}

func contentTypeForPath(relPath, fileName string) string {
	name := strings.ToLower(fileName)
	if name == "" {
		name = strings.ToLower(relPath)
	}
	switch {
	case strings.HasSuffix(name, ".html"), strings.HasSuffix(name, ".htm"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".js"):
		return "application/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".webp"):
		return "image/webp"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	default:
		return "application/octet-stream"
	}
}
