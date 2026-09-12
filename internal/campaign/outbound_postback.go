package campaign

import (
	"context"
	"fmt"
	"strings"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/postback"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	OutboundTriggerConversion = "conversion"
	OutboundTriggerStatus     = "status"
	OutboundTriggerGoal       = "goal"
)

type OutboundPostbackDTO struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Priority         int32  `json:"priority"`
	Enabled          bool   `json:"enabled"`
	Provider         string `json:"provider"`
	URLTemplate      string `json:"url_template"`
	TargetEvent      string `json:"target_event"`
	TriggerKind      string `json:"trigger_kind"`
	TriggerValue     string `json:"trigger_value"`
	TestEventCode    string `json:"test_event_code,omitempty"`
	HasAPIToken      bool   `json:"has_api_token"`
	HasSigningSecret bool   `json:"has_signing_secret"`
	SamplePercent    int32  `json:"sample_percent"`
	DelaySeconds     int32  `json:"delay_seconds"`
}

type OutboundPostbackWriteDTO struct {
	Name          string `json:"name"`
	Priority      int32  `json:"priority"`
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`
	URLTemplate   string `json:"url_template"`
	APIToken      string `json:"api_token"`
	SigningSecret string `json:"signing_secret"`
	TargetEvent   string `json:"target_event"`
	TriggerKind   string `json:"trigger_kind"`
	TriggerValue  string `json:"trigger_value"`
	TestEventCode string `json:"test_event_code"`
	SamplePercent *int32 `json:"sample_percent,omitempty"`
	DelaySeconds  *int32 `json:"delay_seconds,omitempty"`
}

type OutboundPostbackListResponse struct {
	Postbacks []OutboundPostbackDTO `json:"postbacks"`
}

type ReplaceOutboundPostbacksRequest struct {
	Postbacks []OutboundPostbackWriteDTO `json:"postbacks"`
}

type PatchOutboundPostbackRequest struct {
	Name          *string `json:"name,omitempty"`
	Priority      *int32  `json:"priority,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
	Provider      *string `json:"provider,omitempty"`
	URLTemplate   *string `json:"url_template,omitempty"`
	APIToken      *string `json:"api_token,omitempty"`
	SigningSecret *string `json:"signing_secret,omitempty"`
	TargetEvent   *string `json:"target_event,omitempty"`
	TriggerKind   *string `json:"trigger_kind,omitempty"`
	TriggerValue  *string `json:"trigger_value,omitempty"`
	TestEventCode *string `json:"test_event_code,omitempty"`
	SamplePercent *int32  `json:"sample_percent,omitempty"`
	DelaySeconds  *int32  `json:"delay_seconds,omitempty"`
}

type OutboundPostbackService interface {
	ListCampaignOutboundPostbacks(ctx context.Context, campaignID uuid.UUID) ([]OutboundPostbackDTO, error)
	ReplaceCampaignOutboundPostbacks(ctx context.Context, campaignID uuid.UUID, rows []OutboundPostbackWriteDTO) ([]OutboundPostbackDTO, error)
	PatchCampaignOutboundPostback(ctx context.Context, campaignID, postbackID uuid.UUID, patch PatchOutboundPostbackRequest) (OutboundPostbackDTO, error)
	DryRunCampaignOutboundPostback(ctx context.Context, campaignID, postbackID uuid.UUID) (postback.DryRunResult, error)
}

func OutboundPostbackToDTO(row *db.CampaignOutboundPostback) OutboundPostbackDTO {
	if row == nil {
		return OutboundPostbackDTO{}
	}
	return OutboundPostbackDTO{
		ID:               uuid.UUID(row.ID.Bytes).String(),
		Name:             row.Name,
		Priority:         row.Priority,
		Enabled:          row.Enabled,
		Provider:         row.Provider,
		URLTemplate:      row.UrlTemplate,
		TargetEvent:      row.TargetEvent,
		TriggerKind:      row.TriggerKind,
		TriggerValue:     row.TriggerValue,
		TestEventCode:    row.TestEventCode,
		HasAPIToken:      len(row.ApiTokenEncrypted) > 0,
		HasSigningSecret: len(row.SigningSecretEncrypted) > 0,
		SamplePercent:    row.SamplePercent,
		DelaySeconds:     row.DelaySeconds,
	}
}

func NormalizeOutboundPostbacks(rows []OutboundPostbackWriteDTO) ([]OutboundPostbackWriteDTO, error) {
	if len(rows) == 0 {
		return []OutboundPostbackWriteDTO{}, nil
	}
	out := make([]OutboundPostbackWriteDTO, 0, len(rows))
	for i := range rows {
		row, err := normalizeOutboundPostbackWrite(rows[i], i+1)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

func normalizeOutboundPostbackWrite(row OutboundPostbackWriteDTO, index int) (OutboundPostbackWriteDTO, error) {
	provider := strings.ToLower(strings.TrimSpace(row.Provider))
	if provider == "" {
		provider = "webhook"
	}
	if err := validateOutboundProvider(provider, row.URLTemplate); err != nil {
		return OutboundPostbackWriteDTO{}, fmt.Errorf("postback %d: %w", index, err)
	}
	triggerKind := strings.ToLower(strings.TrimSpace(row.TriggerKind))
	if triggerKind == "" {
		triggerKind = OutboundTriggerConversion
	}
	switch triggerKind {
	case OutboundTriggerConversion, OutboundTriggerStatus, OutboundTriggerGoal:
	default:
		return OutboundPostbackWriteDTO{}, fmt.Errorf("postback %d: invalid trigger_kind", index)
	}
	if triggerKind != OutboundTriggerConversion && strings.TrimSpace(row.TriggerValue) == "" {
		return OutboundPostbackWriteDTO{}, fmt.Errorf("postback %d: trigger_value required for %s", index, triggerKind)
	}
	targetEvent := strings.TrimSpace(row.TargetEvent)
	if targetEvent == "" {
		targetEvent = "conversion"
	}
	priority := row.Priority
	if priority < 0 {
		priority = int32(index - 1)
	}
	clamped := outboundWriteSamplePercent(row)
	delaySeconds := outboundWriteDelaySeconds(row)
	return OutboundPostbackWriteDTO{
		Name:          strings.TrimSpace(row.Name),
		Priority:      priority,
		Enabled:       row.Enabled,
		Provider:      provider,
		URLTemplate:   strings.TrimSpace(row.URLTemplate),
		APIToken:      row.APIToken,
		SigningSecret: row.SigningSecret,
		TargetEvent:   targetEvent,
		TriggerKind:   triggerKind,
		TriggerValue:  strings.ToLower(strings.TrimSpace(row.TriggerValue)),
		TestEventCode: strings.TrimSpace(row.TestEventCode),
		SamplePercent: &clamped,
		DelaySeconds:  &delaySeconds,
	}, nil
}

func outboundWriteDelaySeconds(row OutboundPostbackWriteDTO) int32 {
	if row.DelaySeconds != nil {
		return postback.ClampOutboundDelaySeconds(*row.DelaySeconds)
	}
	return 0
}

func outboundWriteSamplePercent(row OutboundPostbackWriteDTO) int32 {
	if row.SamplePercent != nil {
		return clampOutboundSamplePercent(*row.SamplePercent)
	}
	return 100
}

func clampOutboundSamplePercent(value int32) int32 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func validateOutboundProvider(provider, urlTemplate string) error {
	switch provider {
	case "webhook", "facebook", "google", "tiktok", "taboola", "outbrain", "microsoft_ads":
	default:
		return fmt.Errorf("unsupported provider")
	}
	if strings.TrimSpace(urlTemplate) == "" {
		return fmt.Errorf("url_template is required")
	}
	if provider == "google" {
		return postback.ValidateGooglePostbackConfig(urlTemplate)
	}
	return nil
}

func ListCampaignOutboundPostbacks(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) ([]OutboundPostbackDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	rows, err := db.New(pool).ListOutboundPostbacksByCampaign(ctx, domain.ToUUID(campaignID))
	if err != nil {
		return nil, err
	}
	out := make([]OutboundPostbackDTO, 0, len(rows))
	for i := range rows {
		row := db.CampaignOutboundPostbackFromList(rows[i])
		out = append(out, OutboundPostbackToDTO(&row))
	}
	return out, nil
}

func ReplaceCampaignOutboundPostbacks(
	ctx context.Context,
	pool *pgxpool.Pool,
	encryptionKey []byte,
	campaignID uuid.UUID,
	rows []OutboundPostbackWriteDTO,
) ([]OutboundPostbackDTO, error) {
	if pool == nil {
		return nil, errServiceUnavailable()
	}
	normalized, err := NormalizeOutboundPostbacks(rows)
	if err != nil {
		return nil, err
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	saved, err := ReplaceCampaignOutboundPostbacksTx(ctx, db.New(tx), encryptionKey, campaignID, normalized)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return saved, nil
}

func ReplaceCampaignOutboundPostbacksTx(
	ctx context.Context,
	q *db.Queries,
	encryptionKey []byte,
	campaignID uuid.UUID,
	rows []OutboundPostbackWriteDTO,
) ([]OutboundPostbackDTO, error) {
	if q == nil {
		return nil, errServiceUnavailable()
	}
	normalized, err := NormalizeOutboundPostbacks(rows)
	if err != nil {
		return nil, err
	}
	if err := q.DeleteOutboundPostbacksByCampaign(ctx, domain.ToUUID(campaignID)); err != nil {
		return nil, err
	}
	out := make([]OutboundPostbackDTO, 0, len(normalized))
	for i := range normalized {
		row := normalized[i]
		if postback.ProviderRequiresToken(row.Provider) && strings.TrimSpace(row.APIToken) == "" {
			return nil, fmt.Errorf("postback %d: api_token is required for CAPI providers", i+1)
		}
		id, err := uuid.NewV7()
		if err != nil {
			return nil, err
		}
		encToken, encSecret, err := encryptOutboundSecrets(row.APIToken, row.SigningSecret, encryptionKey)
		if err != nil {
			return nil, err
		}
		inserted, err := q.InsertOutboundPostback(ctx, db.InsertOutboundPostbackParams{
			ID:                     domain.ToUUID(id),
			CampaignID:             domain.ToUUID(campaignID),
			Name:                   row.Name,
			Priority:               row.Priority,
			Enabled:                row.Enabled,
			Provider:               row.Provider,
			UrlTemplate:            row.URLTemplate,
			ApiTokenEncrypted:      encToken,
			TargetEvent:            row.TargetEvent,
			TriggerKind:            row.TriggerKind,
			TriggerValue:           row.TriggerValue,
			TestEventCode:          row.TestEventCode,
			SigningSecretEncrypted: encSecret,
			SamplePercent:          outboundWriteSamplePercent(row),
			DelaySeconds:           outboundWriteDelaySeconds(row),
		})
		if err != nil {
			return nil, fmt.Errorf("insert outbound postback: %w", err)
		}
		insertedRow := db.CampaignOutboundPostbackFromInsert(inserted)
		out = append(out, OutboundPostbackToDTO(&insertedRow))
	}
	return out, nil
}

func PatchCampaignOutboundPostback(
	ctx context.Context,
	pool *pgxpool.Pool,
	encryptionKey []byte,
	campaignID, postbackID uuid.UUID,
	patch PatchOutboundPostbackRequest,
) (OutboundPostbackDTO, error) {
	if pool == nil {
		return OutboundPostbackDTO{}, errServiceUnavailable()
	}
	q := db.New(pool)
	current, err := q.GetOutboundPostback(ctx, db.GetOutboundPostbackParams{
		ID:         domain.ToUUID(postbackID),
		CampaignID: domain.ToUUID(campaignID),
	})
	if err != nil {
		return OutboundPostbackDTO{}, err
	}
	currentRow := db.CampaignOutboundPostbackFromGet(current)
	next := outboundWriteFromRow(&currentRow)
	if patch.Name != nil {
		next.Name = strings.TrimSpace(*patch.Name)
	}
	if patch.Priority != nil {
		next.Priority = *patch.Priority
	}
	if patch.Enabled != nil {
		next.Enabled = *patch.Enabled
	}
	if patch.Provider != nil {
		next.Provider = strings.TrimSpace(*patch.Provider)
	}
	if patch.URLTemplate != nil {
		next.URLTemplate = strings.TrimSpace(*patch.URLTemplate)
	}
	if patch.APIToken != nil {
		next.APIToken = *patch.APIToken
	}
	if patch.SigningSecret != nil {
		next.SigningSecret = *patch.SigningSecret
	}
	if patch.TargetEvent != nil {
		next.TargetEvent = strings.TrimSpace(*patch.TargetEvent)
	}
	if patch.TriggerKind != nil {
		next.TriggerKind = strings.TrimSpace(*patch.TriggerKind)
	}
	if patch.TriggerValue != nil {
		next.TriggerValue = strings.TrimSpace(*patch.TriggerValue)
	}
	if patch.TestEventCode != nil {
		next.TestEventCode = strings.TrimSpace(*patch.TestEventCode)
	}
	if patch.SamplePercent != nil {
		clamped := clampOutboundSamplePercent(*patch.SamplePercent)
		next.SamplePercent = &clamped
	}
	if patch.DelaySeconds != nil {
		clamped := postback.ClampOutboundDelaySeconds(*patch.DelaySeconds)
		next.DelaySeconds = &clamped
	}
	normalized, err := NormalizeOutboundPostbacks([]OutboundPostbackWriteDTO{next})
	if err != nil {
		return OutboundPostbackDTO{}, err
	}
	next = normalized[0]
	encToken := current.ApiTokenEncrypted
	encSecret := current.SigningSecretEncrypted
	if next.APIToken != "" {
		encToken, encSecret, err = encryptOutboundSecrets(next.APIToken, next.SigningSecret, encryptionKey)
		if err != nil {
			return OutboundPostbackDTO{}, err
		}
	} else if next.SigningSecret != "" {
		encSecret, err = encryptSigningSecret(next.SigningSecret, encryptionKey)
		if err != nil {
			return OutboundPostbackDTO{}, err
		}
	}
	if postback.ProviderRequiresToken(next.Provider) && len(encToken) == 0 {
		return OutboundPostbackDTO{}, fmt.Errorf("api_token is required for CAPI providers")
	}
	updated, err := q.UpdateOutboundPostback(ctx, db.UpdateOutboundPostbackParams{
		ID:                     domain.ToUUID(postbackID),
		CampaignID:             domain.ToUUID(campaignID),
		Name:                   next.Name,
		Priority:               next.Priority,
		Enabled:                next.Enabled,
		Provider:               next.Provider,
		UrlTemplate:            next.URLTemplate,
		ApiTokenEncrypted:      encToken,
		TargetEvent:            next.TargetEvent,
		TriggerKind:            next.TriggerKind,
		TriggerValue:           next.TriggerValue,
		TestEventCode:          next.TestEventCode,
		SigningSecretEncrypted: encSecret,
		SamplePercent:          outboundWriteSamplePercent(next),
		DelaySeconds:           outboundWriteDelaySeconds(next),
	})
	if err != nil {
		return OutboundPostbackDTO{}, err
	}
	updatedRow := db.CampaignOutboundPostbackFromUpdate(updated)
	return OutboundPostbackToDTO(&updatedRow), nil
}

func DryRunCampaignOutboundPostback(
	ctx context.Context,
	pool *pgxpool.Pool,
	encryptionKey []byte,
	campaignID, postbackID uuid.UUID,
) (postback.DryRunResult, error) {
	if pool == nil {
		return postback.DryRunResult{}, errServiceUnavailable()
	}
	row, err := db.New(pool).GetOutboundPostback(ctx, db.GetOutboundPostbackParams{
		ID:         domain.ToUUID(postbackID),
		CampaignID: domain.ToUUID(campaignID),
	})
	if err != nil {
		return postback.DryRunResult{}, err
	}
	token, err := decryptOutboundToken(row.ApiTokenEncrypted, encryptionKey)
	if err != nil {
		return postback.DryRunResult{}, err
	}
	record := db.CampaignOutboundPostbackFromGet(row)
	return postback.DryRunConfig(ctx, record.Provider, record.UrlTemplate, token, record.TargetEvent, record.TestEventCode, campaignID), nil
}

func CloneCampaignOutboundPostbacks(ctx context.Context, tx pgx.Tx, destID, sourceID uuid.UUID) error {
	if tx == nil {
		return errServiceUnavailable()
	}
	_, err := tx.Exec(ctx, `
INSERT INTO campaign_outbound_postbacks (
    id, campaign_id, name, priority, enabled, provider, url_template, api_token_encrypted,
    target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted, sample_percent, delay_seconds
)
SELECT gen_random_uuid(), $1, name, priority, enabled, provider, url_template, api_token_encrypted,
       target_event, trigger_kind, trigger_value, test_event_code, signing_secret_encrypted, sample_percent, delay_seconds
FROM campaign_outbound_postbacks
WHERE campaign_id = $2`, destID, sourceID)
	return err
}

func outboundWriteFromRow(row *db.CampaignOutboundPostback) OutboundPostbackWriteDTO {
	if row == nil {
		return OutboundPostbackWriteDTO{}
	}
	samplePercent := row.SamplePercent
	delaySeconds := row.DelaySeconds
	return OutboundPostbackWriteDTO{
		Name:          row.Name,
		Priority:      row.Priority,
		Enabled:       row.Enabled,
		Provider:      row.Provider,
		URLTemplate:   row.UrlTemplate,
		TargetEvent:   row.TargetEvent,
		TriggerKind:   row.TriggerKind,
		TriggerValue:  row.TriggerValue,
		TestEventCode: row.TestEventCode,
		SamplePercent: &samplePercent,
		DelaySeconds:  &delaySeconds,
	}
}

func encryptOutboundSecrets(apiToken, signingSecret string, encryptionKey []byte) ([]byte, []byte, error) {
	key := outboundEncryptionKey(encryptionKey)
	encToken := []byte{}
	if strings.TrimSpace(apiToken) != "" {
		token, err := postback.EncryptAESGCM([]byte(apiToken), key)
		if err != nil {
			return nil, nil, err
		}
		encToken = token
	}
	encSecret, err := encryptSigningSecret(signingSecret, encryptionKey)
	if err != nil {
		return nil, nil, err
	}
	return encToken, encSecret, nil
}

func encryptSigningSecret(signingSecret string, encryptionKey []byte) ([]byte, error) {
	if strings.TrimSpace(signingSecret) == "" {
		return []byte{}, nil
	}
	return postback.EncryptAESGCM([]byte(signingSecret), outboundEncryptionKey(encryptionKey))
}

func decryptOutboundToken(enc []byte, encryptionKey []byte) (string, error) {
	if len(enc) == 0 {
		return "", nil
	}
	plain, err := postback.DecryptAESGCM(enc, outboundEncryptionKey(encryptionKey))
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func outboundEncryptionKey(key []byte) []byte {
	if len(key) == 0 {
		return []byte("postback-encryption-secret-key32")
	}
	return key
}

func ExportOutboundPostbacksFromRows(rows []db.CampaignOutboundPostback) []CampaignExportOutboundPostback {
	out := make([]CampaignExportOutboundPostback, 0, len(rows))
	for i := range rows {
		row := rows[i]
		out = append(out, CampaignExportOutboundPostback{
			Name:          row.Name,
			Priority:      row.Priority,
			Enabled:       row.Enabled,
			Provider:      row.Provider,
			URLTemplate:   row.UrlTemplate,
			TargetEvent:   row.TargetEvent,
			TriggerKind:   row.TriggerKind,
			TriggerValue:  row.TriggerValue,
			TestEventCode: row.TestEventCode,
			SamplePercent: row.SamplePercent,
			DelaySeconds:  row.DelaySeconds,
		})
	}
	return out
}

func OutboundPostbackWritesFromExport(rows []CampaignExportOutboundPostback) ([]OutboundPostbackWriteDTO, error) {
	if len(rows) == 0 {
		return []OutboundPostbackWriteDTO{}, nil
	}
	out := make([]OutboundPostbackWriteDTO, 0, len(rows))
	for i := range rows {
		row := rows[i]
		samplePercent := row.SamplePercent
		delaySeconds := row.DelaySeconds
		out = append(out, OutboundPostbackWriteDTO{
			Name:          row.Name,
			Priority:      row.Priority,
			Enabled:       row.Enabled,
			Provider:      row.Provider,
			URLTemplate:   row.URLTemplate,
			TargetEvent:   row.TargetEvent,
			TriggerKind:   row.TriggerKind,
			TriggerValue:  row.TriggerValue,
			TestEventCode: row.TestEventCode,
			SamplePercent: &samplePercent,
			DelaySeconds:  &delaySeconds,
		})
	}
	return NormalizeOutboundPostbacks(out)
}
