package controlplane

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

type HostedEditorFileDTO struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Editable bool   `json:"editable"`
}

type HostedEditorStateDTO struct {
	LanderID            uuid.UUID             `json:"lander_id"`
	Name                string                `json:"name"`
	DraftVersion        int                   `json:"draft_version"`
	PublishedVersion    int                   `json:"published_version"`
	HasUnpublishedDraft bool                  `json:"has_unpublished_draft"`
	Files               []HostedEditorFileDTO `json:"files"`
	PreviewURL          string                `json:"preview_url,omitempty"`
}

type HostedEditorFileBodyDTO struct {
	Content string `json:"content"`
}

type HostedEditorSaveResultDTO struct {
	DraftVersion        int  `json:"draft_version"`
	HasUnpublishedDraft bool `json:"has_unpublished_draft"`
}

type HostedEditorPublishRequest struct {
	Version int `json:"version"`
}

type landerVersionRow struct {
	draftVersion     int
	publishedVersion int
}

func (s *Service) landerPreviewSecret() []byte {
	if s == nil || s.cfg == nil {
		return nil
	}
	if raw := strings.TrimSpace(s.cfg.LanderPreviewSecret); raw != "" {
		return []byte(raw)
	}
	if len(s.cfg.ConsentHMACSecret) > 0 {
		return []byte(s.cfg.ConsentHMACSecret)
	}
	return nil
}

func (s *Service) loadLanderVersionRow(ctx context.Context, landerID uuid.UUID) (landerVersionRow, error) {
	var row landerVersionRow
	err := s.pool.QueryRow(ctx, `
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

func (s *Service) GetHostedEditorState(ctx context.Context, landerID uuid.UUID) (HostedEditorStateDTO, error) {
	if s == nil || s.pool == nil {
		return HostedEditorStateDTO{}, fmt.Errorf("service unavailable")
	}
	st := s.hostedLanderStore()
	if st == nil {
		return HostedEditorStateDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	versions, err := s.loadLanderVersionRow(ctx, landerID)
	if err != nil {
		return HostedEditorStateDTO{}, err
	}
	var name string
	if err := s.pool.QueryRow(ctx, `SELECT name FROM landers WHERE id = $1`, landerID).Scan(&name); err != nil {
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
	secret := s.landerPreviewSecret()
	if len(secret) > 0 {
		token, err := landerhost.MintPreviewToken(secret, landerID, versions.draftVersion, time.Now())
		if err == nil {
			base := s.landerPublicBase(ctx)
			if base == "" && s.cfg != nil {
				base = strings.TrimRight(strings.TrimSpace(s.cfg.ManagementURL), "/")
			}
			dto.PreviewURL = landerhost.PreviewURL(base, landerID, token)
		}
	}
	return dto, nil
}

func (s *Service) ReadHostedEditorFile(ctx context.Context, landerID uuid.UUID, relPath string) (HostedEditorFileBodyDTO, error) {
	if s == nil || s.pool == nil {
		return HostedEditorFileBodyDTO{}, fmt.Errorf("service unavailable")
	}
	st := s.hostedLanderStore()
	if st == nil {
		return HostedEditorFileBodyDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	versions, err := s.loadLanderVersionRow(ctx, landerID)
	if err != nil {
		return HostedEditorFileBodyDTO{}, err
	}
	raw, err := st.ReadVersionFile(landerID, versions.draftVersion, relPath)
	if err != nil {
		return HostedEditorFileBodyDTO{}, err
	}
	return HostedEditorFileBodyDTO{Content: string(raw)}, nil
}

func (s *Service) SaveHostedEditorFile(ctx context.Context, landerID uuid.UUID, relPath, content string) (HostedEditorSaveResultDTO, error) {
	if s == nil || s.pool == nil {
		return HostedEditorSaveResultDTO{}, fmt.Errorf("service unavailable")
	}
	st := s.hostedLanderStore()
	if st == nil {
		return HostedEditorSaveResultDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	if int64(len(content)) > landerhost.DefaultMaxEditorFileBytes {
		return HostedEditorSaveResultDTO{}, fmt.Errorf("file exceeds editor size limit")
	}
	versions, err := s.loadLanderVersionRow(ctx, landerID)
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

	tx, err := s.pool.Begin(ctx)
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

func (s *Service) PublishHostedDraft(ctx context.Context, landerID uuid.UUID, version int) (LanderDTO, error) {
	if s == nil || s.pool == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	st := s.hostedLanderStore()
	if st == nil {
		return LanderDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	versions, err := s.loadLanderVersionRow(ctx, landerID)
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

	tx, err := s.pool.Begin(ctx)
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
	dto.HostedURL = landerhost.PublicURL(s.landerPublicBase(ctx), landerID)
	return dto, nil
}

func (s *Service) ServeHostedPreviewFile(ctx context.Context, landerID uuid.UUID, version int, relPath, token string) (io.ReadCloser, string, error) {
	st := s.hostedLanderStore()
	if st == nil {
		return nil, "", fmt.Errorf("hosted lander store is not configured")
	}
	secret := s.landerPreviewSecret()
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
