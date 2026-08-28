package controlplane

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"ad-event-processor/internal/flow"
	"ad-event-processor/internal/outbox"
	"ad-event-processor/pkg/landerhost"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var _ flow.Host = (*Service)(nil)
var _ flow.HostedLanderHost = (*Service)(nil)
var _ flow.PathRefChecker = (*Service)(nil)

func (s *Service) FlowStore() *flow.Store {
	if s == nil {
		return nil
	}
	if s.flowStore == nil {
		s.flowStore = flow.NewStore(s.pool, s)
	}
	return s.flowStore
}

func (s *Service) LanderPublicBase(ctx context.Context) string {
	return s.landerPublicBase(ctx)
}

func (s *Service) PublishFlowReload(ctx context.Context) error {
	if s == nil {
		return nil
	}
	channel := ""
	if s.cfg != nil && strings.TrimSpace(s.cfg.FlowReloadChannel) != "" {
		channel = strings.TrimSpace(s.cfg.FlowReloadChannel)
	}
	return outbox.PublishFlowReload(ctx, s.redisShards, channel)
}

func (s *Service) CreateLander(ctx context.Context, req CreateLanderRequest) (LanderDTO, error) {
	return s.FlowStore().CreateLander(ctx, req)
}

func (s *Service) ListLanders(ctx context.Context) ([]LanderDTO, error) {
	return s.FlowStore().ListLanders(ctx)
}

func (s *Service) CreateOffer(ctx context.Context, req CreateOfferRequest) (OfferDTO, error) {
	return s.FlowStore().CreateOffer(ctx, req)
}

func (s *Service) ListOffers(ctx context.Context) ([]OfferDTO, error) {
	return s.FlowStore().ListOffers(ctx)
}

func (s *Service) CreateFlow(ctx context.Context, req CreateFlowRequest) (FlowDTO, error) {
	return s.FlowStore().CreateFlow(ctx, req)
}

func (s *Service) ListFlows(ctx context.Context) ([]FlowDTO, error) {
	return s.FlowStore().ListFlows(ctx)
}

func (s *Service) GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error) {
	return s.FlowStore().GetFlow(ctx, flowID)
}

func (s *Service) UpdateFlow(ctx context.Context, flowID uuid.UUID, req UpdateFlowRequest) (FlowDTO, error) {
	return s.FlowStore().UpdateFlow(ctx, flowID, req)
}

func (s *Service) AssignCampaignFlow(ctx context.Context, campaignID, flowID uuid.UUID) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if campaignID == uuid.Nil {
		return fmt.Errorf("campaign id required")
	}
	if flowID != uuid.Nil {
		var one int
		err := s.pool.QueryRow(ctx, `SELECT 1 FROM flows WHERE id = $1`, flowID).Scan(&one)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("flow not found")
			}
			return err
		}
	}
	tag, err := s.pool.Exec(ctx, `UPDATE campaigns SET flow_id = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, campaignID, flowIDOrNil(flowID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("campaign not found")
	}
	_ = s.publishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func flowIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func (s *Service) campaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error) {
	if s == nil || s.pool == nil {
		return "", fmt.Errorf("service unavailable")
	}
	var flowID pgtype.UUID
	err := s.pool.QueryRow(ctx, `SELECT flow_id FROM campaigns WHERE id = $1`, campaignID).Scan(&flowID)
	if err != nil {
		return "", err
	}
	if !flowID.Valid {
		return "", nil
	}
	return uuid.UUID(flowID.Bytes).String(), nil
}

func (s *Service) ValidateCampaignFlowPaths(ctx context.Context, paths []FlowPathDTO) error {
	return flow.ValidatePathRefs(ctx, s, paths)
}

func (s *Service) ValidateLanderIDs(ctx context.Context, ids []uuid.UUID) error {
	return flow.ValidateLanderIDsPG(ctx, s.pool, ids)
}

func (s *Service) ValidateOfferIDs(ctx context.Context, ids []uuid.UUID) error {
	return flow.ValidateOfferIDsPG(ctx, s.pool, ids)
}

func (s *Service) HostedLanderPool() *pgxpool.Pool {
	if s == nil {
		return nil
	}
	return s.pool
}

func (s *Service) HostedLanderStore() *landerhost.Store {
	if s == nil {
		return nil
	}
	if s.landerStore == nil {
		return s.initLanderStore()
	}
	return s.landerStore
}

func (s *Service) LanderPreviewSecret() []byte {
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

func (s *Service) LanderManagementURL() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	return s.cfg.ManagementURL
}

func (s *Service) LanderMaxZipBytes() int64 {
	if s == nil || s.cfg == nil {
		return 0
	}
	return s.cfg.LanderMaxZipBytes
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

func (s *Service) UploadHostedLanderZip(ctx context.Context, landerID uuid.UUID, zipReader io.ReaderAt, zipSize int64) (LanderDTO, error) {
	return flow.UploadHostedLanderZip(ctx, s, landerID, zipReader, zipSize)
}

func (s *Service) ServeHostedLanderFile(ctx context.Context, landerID uuid.UUID, relPath string) (io.ReadCloser, string, error) {
	return flow.ServeHostedLanderFile(ctx, s, landerID, relPath)
}

func (s *Service) GetHostedEditorState(ctx context.Context, landerID uuid.UUID) (HostedEditorStateDTO, error) {
	return flow.GetHostedEditorState(ctx, s, landerID)
}

func (s *Service) ReadHostedEditorFile(ctx context.Context, landerID uuid.UUID, relPath string) (HostedEditorFileBodyDTO, error) {
	return flow.ReadHostedEditorFile(ctx, s, landerID, relPath)
}

func (s *Service) SaveHostedEditorFile(ctx context.Context, landerID uuid.UUID, relPath, content string) (HostedEditorSaveResultDTO, error) {
	return flow.SaveHostedEditorFile(ctx, s, landerID, relPath, content)
}

func (s *Service) PublishHostedDraft(ctx context.Context, landerID uuid.UUID, version int) (LanderDTO, error) {
	return flow.PublishHostedDraft(ctx, s, landerID, version)
}

func (s *Service) ServeHostedPreviewFile(ctx context.Context, landerID uuid.UUID, version int, relPath, token string) (io.ReadCloser, string, error) {
	return flow.ServeHostedPreviewFile(ctx, s, landerID, version, relPath, token)
}
