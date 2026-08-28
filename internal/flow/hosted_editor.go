package flow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type landerVersionRow struct {
	draftVersion     int
	publishedVersion int
}

func loadLanderVersionRow(ctx context.Context, host HostedLanderHost, landerID uuid.UUID) (landerVersionRow, error) {
	var row landerVersionRow
	pool := host.HostedLanderPool()
	if pool == nil {
		return row, fmt.Errorf("service unavailable")
	}
	err := pool.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT MAX(la.version) FROM lander_assets la WHERE la.lander_id = l.id), 0),
			COALESCE(pub.version, 0)
		FROM landers l
		LEFT JOIN lander_assets pub ON pub.id = l.hosted_asset_id
		WHERE l.id = $1`, landerID).Scan(&row.draftVersion, &row.publishedVersion)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return row, fmt.Errorf("lander not found")
		}
		return row, err
	}
	if row.draftVersion <= 0 {
		return row, fmt.Errorf("lander has no hosted files")
	}
	return row, nil
}

func GetHostedEditorState(ctx context.Context, host HostedLanderHost, landerID uuid.UUID) (HostedEditorStateDTO, error) {
	pool := host.HostedLanderPool()
	if pool == nil {
		return HostedEditorStateDTO{}, fmt.Errorf("service unavailable")
	}
	st := host.HostedLanderStore()
	if st == nil {
		return HostedEditorStateDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	versions, err := loadLanderVersionRow(ctx, host, landerID)
	if err != nil {
		return HostedEditorStateDTO{}, err
	}
	var name string
	if err := pool.QueryRow(ctx, `SELECT name FROM landers WHERE id = $1`, landerID).Scan(&name); err != nil {
		return HostedEditorStateDTO{}, err
	}
	files, err := st.ListVersionFiles(landerID, versions.draftVersion)
	if err != nil {
		return HostedEditorStateDTO{}, err
	}
	dto := HostedEditorStateDTO{
		LanderID:            landerID,
		Name:                name,
		DraftVersion:        versions.draftVersion,
		PublishedVersion:    versions.publishedVersion,
		HasUnpublishedDraft: versions.draftVersion > versions.publishedVersion,
		Files:               make([]HostedEditorFileDTO, 0, len(files)),
	}
	for _, f := range files {
		dto.Files = append(dto.Files, HostedEditorFileDTO(f))
	}
	secret := host.LanderPreviewSecret()
	if len(secret) > 0 {
		token, err := landerhost.MintPreviewToken(secret, landerID, versions.draftVersion, time.Now())
		if err == nil {
			base := host.LanderPublicBase(ctx)
			if base == "" {
				base = strings.TrimRight(strings.TrimSpace(host.LanderManagementURL()), "/")
			}
			dto.PreviewURL = landerhost.PreviewURL(base, landerID, token)
		}
	}
	return dto, nil
}

func ReadHostedEditorFile(ctx context.Context, host HostedLanderHost, landerID uuid.UUID, relPath string) (HostedEditorFileBodyDTO, error) {
	if host.HostedLanderPool() == nil {
		return HostedEditorFileBodyDTO{}, fmt.Errorf("service unavailable")
	}
	st := host.HostedLanderStore()
	if st == nil {
		return HostedEditorFileBodyDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	versions, err := loadLanderVersionRow(ctx, host, landerID)
	if err != nil {
		return HostedEditorFileBodyDTO{}, err
	}
	raw, err := st.ReadVersionFile(landerID, versions.draftVersion, relPath)
	if err != nil {
		return HostedEditorFileBodyDTO{}, err
	}
	return HostedEditorFileBodyDTO{Content: string(raw)}, nil
}

func SaveHostedEditorFile(ctx context.Context, host HostedLanderHost, landerID uuid.UUID, relPath, content string) (HostedEditorSaveResultDTO, error) {
	pool := host.HostedLanderPool()
	if pool == nil {
		return HostedEditorSaveResultDTO{}, fmt.Errorf("service unavailable")
	}
	st := host.HostedLanderStore()
	if st == nil {
		return HostedEditorSaveResultDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	if int64(len(content)) > landerhost.DefaultMaxEditorFileBytes {
		return HostedEditorSaveResultDTO{}, fmt.Errorf("file exceeds editor size limit")
	}
	versions, err := loadLanderVersionRow(ctx, host, landerID)
	if err != nil {
		return HostedEditorSaveResultDTO{}, err
	}
	nextVersion := versions.draftVersion + 1
	if err := st.CloneVersion(landerID, versions.draftVersion, nextVersion); err != nil {
		return HostedEditorSaveResultDTO{}, err
	}
	if err := st.WriteVersionTextFile(landerID, nextVersion, relPath, []byte(content)); err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return HostedEditorSaveResultDTO{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return HostedEditorSaveResultDTO{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	files, err := st.ListVersionFiles(landerID, nextVersion)
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return HostedEditorSaveResultDTO{}, err
	}
	var assetID uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO lander_assets (lander_id, version, entry_count)
		VALUES ($1, $2, $3)
		RETURNING id`, landerID, nextVersion, len(files)).Scan(&assetID)
	if err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return HostedEditorSaveResultDTO{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		_ = os.RemoveAll(st.VersionDir(landerID, nextVersion))
		return HostedEditorSaveResultDTO{}, err
	}
	published := versions.publishedVersion
	if published <= 0 {
		published = versions.draftVersion
	}
	return HostedEditorSaveResultDTO{
		DraftVersion:        nextVersion,
		HasUnpublishedDraft: nextVersion > published,
	}, nil
}

func PublishHostedDraft(ctx context.Context, host HostedLanderHost, landerID uuid.UUID, version int) (LanderDTO, error) {
	pool := host.HostedLanderPool()
	if pool == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	st := host.HostedLanderStore()
	if st == nil {
		return LanderDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	versions, err := loadLanderVersionRow(ctx, host, landerID)
	if err != nil {
		return LanderDTO{}, err
	}
	publishVersion := version
	if publishVersion <= 0 {
		publishVersion = versions.draftVersion
	}
	if publishVersion <= 0 {
		return LanderDTO{}, fmt.Errorf("version is required")
	}
	if err := st.PublishVersion(landerID, publishVersion); err != nil {
		return LanderDTO{}, err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return LanderDTO{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var assetID uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT id FROM lander_assets WHERE lander_id = $1 AND version = $2`,
		landerID, publishVersion).Scan(&assetID)
	if err != nil {
		return LanderDTO{}, fmt.Errorf("draft version not found")
	}

	var dto LanderDTO
	err = tx.QueryRow(ctx, `
		UPDATE landers
		SET hosted_asset_id = $2, url = ''
		WHERE id = $1
		RETURNING id, name, COALESCE(url, ''), hosted_asset_id, created_at`,
		landerID, assetID).Scan(&dto.ID, &dto.Name, &dto.URL, &dto.HostedAssetID, &dto.CreatedAt)
	if err != nil {
		return LanderDTO{}, err
	}
	payload, err := json.Marshal(landerPublishedPayload{LanderID: landerID.String()})
	if err != nil {
		return LanderDTO{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events (event_type, payload, status)
		VALUES ('LANDER_PUBLISHED', $1, 'PENDING')`, payload); err != nil {
		return LanderDTO{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return LanderDTO{}, err
	}
	dto.HostedURL = landerhost.PublicURL(host.LanderPublicBase(ctx), landerID)
	return dto, nil
}

func ServeHostedPreviewFile(ctx context.Context, host HostedLanderHost, landerID uuid.UUID, version int, relPath, token string) (io.ReadCloser, string, error) {
	st := host.HostedLanderStore()
	if st == nil {
		return nil, "", fmt.Errorf("hosted lander store is not configured")
	}
	secret := host.LanderPreviewSecret()
	ver, ok := landerhost.VerifyPreviewToken(secret, token, landerID, time.Now())
	if !ok {
		return nil, "", fmt.Errorf("invalid preview token")
	}
	if version > 0 && ver != version {
		return nil, "", fmt.Errorf("invalid preview token")
	}
	rc, info, err := st.OpenPreviewFile(landerID, ver, relPath)
	if err != nil {
		return nil, "", err
	}
	return rc, contentTypeForPath(relPath, info.Name()), nil
}
