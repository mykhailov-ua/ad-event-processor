package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const flowReloadChannel = "flow:reload"

type landerPublishedPayload struct {
	LanderID string `json:"lander_id"`
}

func (s *Service) hostedLanderStore() *landerhost.Store {
	if s == nil {
		return nil
	}
	if s.landerStore == nil {
		return s.initLanderStore()
	}
	return s.landerStore
}

func (s *Service) landerPublicBase(ctx context.Context) string {
	if s == nil {
		return ""
	}
	if base := strings.TrimSpace(s.cfg.LanderPublicBaseURL); base != "" {
		return strings.TrimRight(base, "/")
	}
	cfg, _, err := s.GetPlatformConfig(ctx)
	if err == nil && strings.TrimSpace(cfg.TrackingDomain) != "" {
		return "https://" + strings.TrimSpace(cfg.TrackingDomain)
	}
	return ""
}

func (s *Service) UploadHostedLanderZip(ctx context.Context, landerID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (LanderDTO, error) {
	if s == nil || s.pool == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	st := s.hostedLanderStore()
	if st == nil {
		return LanderDTO{}, fmt.Errorf("hosted lander store is not configured")
	}
	if landerID == uuid.Nil {
		return LanderDTO{}, fmt.Errorf("lander id required")
	}
	maxBytes := s.cfg.LanderMaxZipBytes
	if maxBytes <= 0 {
		maxBytes = landerhost.DefaultMaxZipBytes
	}
	if zipSize <= 0 || zipSize > maxBytes {
		return LanderDTO{}, fmt.Errorf("zip size must be between 1 and %d bytes", maxBytes)
	}

	tx, err := s.pool.Begin(ctx)
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

	tx2, err := s.pool.Begin(ctx)
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

	dto.HostedURL = landerhost.PublicURL(s.landerPublicBase(ctx), landerID)
	return dto, nil
}

func (s *Service) ServeHostedLanderFile(ctx context.Context, landerID uuid.UUID, relPath string) (io.ReadCloser, string, error) {
	st := s.hostedLanderStore()
	if st == nil {
		return nil, "", fmt.Errorf("hosted lander store is not configured")
	}
	if landerID == uuid.Nil {
		return nil, "", fmt.Errorf("lander id required")
	}
	if s.pool != nil {
		var assetID *uuid.UUID
		err := s.pool.QueryRow(ctx, `SELECT hosted_asset_id FROM landers WHERE id = $1`, landerID).Scan(&assetID)
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

func (s *Service) initLanderStore() *landerhost.Store {
	if s == nil || s.landerStore != nil || s.cfg == nil {
		return s.landerStore
	}
	root := strings.TrimSpace(s.cfg.LanderStoreRoot)
	if root == "" {
		return nil
	}
	st, err := landerhost.NewStore(root)
	if err != nil {
		return nil
	}
	s.landerStore = st
	return st
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

func publishFlowReload(ctx context.Context, redisShards []redis.UniversalClient, channel string) error {
	if channel == "" {
		channel = flowReloadChannel
	}
	if len(redisShards) == 0 || redisShards[0] == nil {
		return nil
	}
	return redisShards[0].Publish(ctx, channel, "1").Err()
}

func (w *OutboxWorker) handleLanderPublished(ctx context.Context, payload []byte) error {
	if w == nil || w.svc == nil {
		return fmt.Errorf("outbox worker unavailable")
	}
	if _, err := coldpath.UnmarshalStrict[landerPublishedPayload](payload); err != nil {
		return err
	}
	channel := flowReloadChannel
	if w.svc.cfg != nil && strings.TrimSpace(w.svc.cfg.FlowReloadChannel) != "" {
		channel = strings.TrimSpace(w.svc.cfg.FlowReloadChannel)
	}
	return publishFlowReload(ctx, w.svc.redisShards, channel)
}
