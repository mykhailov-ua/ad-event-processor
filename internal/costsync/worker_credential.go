package costsync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/costsync/provider"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/postback"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (w *Worker) DecryptCredential(row db.CostSyncCredential) (Credential, error) {
	cred := Credential{
		CustomerID: row.CustomerID.Bytes,
		Network:    row.Network,
		AccountID:  row.AccountID,
	}
	if len(row.AccessTokenEncrypted) > 0 {
		b, err := postback.DecryptAESGCM(row.AccessTokenEncrypted, w.encryptionKey)
		if err != nil {
			return cred, err
		}
		cred.AccessToken = string(b)
	}
	if len(row.RefreshTokenEncrypted) > 0 {
		b, err := postback.DecryptAESGCM(row.RefreshTokenEncrypted, w.encryptionKey)
		if err != nil {
			return cred, err
		}
		cred.RefreshToken = string(b)
	}
	if len(row.ApiKeyEncrypted) > 0 {
		b, err := postback.DecryptAESGCM(row.ApiKeyEncrypted, w.encryptionKey)
		if err != nil {
			return cred, err
		}
		cred.APIKey = string(b)
	}
	if len(row.ExtraConfig) > 0 {
		if err := json.Unmarshal(row.ExtraConfig, &cred.ExtraConfig); err != nil {
			return cred, fmt.Errorf("parse extra_config for network=%s account=%s: %w", row.Network, row.AccountID, err)
		}
	}
	if row.TokenExpiresAt.Valid {
		cred.ExpiresAt = row.TokenExpiresAt.Time
	}
	cred.SyncIntervalMinutes = int(row.SyncIntervalMinutes)
	cred.TokenMapping = ParseTokenMapping(row.TokenMapping)
	return cred, nil
}

func (w *Worker) MaybeRefreshToken(ctx context.Context, network string, row db.CostSyncCredential, cred *Credential) error {
	if !cred.ExpiresAt.IsZero() && time.Until(cred.ExpiresAt) > 5*time.Minute {
		return nil
	}

	var (
		token      string
		newRefresh string
		expires    time.Time
		err        error
	)
	switch network {
	case "facebook":
		if w.oauth.MetaAppID == "" || w.oauth.MetaAppSecret == "" {
			return nil
		}
		token, expires, err = provider.RefreshMetaOAuth(ctx, w.httpClient, w.oauth.MetaAppID, w.oauth.MetaAppSecret, *cred)
	case "google":
		if w.oauth.GoogleClientID == "" || w.oauth.GoogleClientSecret == "" {
			return nil
		}
		token, expires, err = provider.RefreshGoogleOAuth(ctx, w.httpClient, w.oauth.GoogleClientID, w.oauth.GoogleClientSecret, *cred)
	case "tiktok":
		if w.oauth.TikTokAppID == "" || w.oauth.TikTokAppSecret == "" {
			return nil
		}
		token, newRefresh, expires, err = provider.RefreshTikTokOAuth(ctx, w.httpClient, w.networkBaseURL["tiktok"], w.oauth.TikTokAppID, w.oauth.TikTokAppSecret, *cred)
	case "revcontent":
		token, expires, err = provider.RefreshRevcontentOAuth(ctx, w.httpClient, w.networkBaseURL["revcontent"], *cred)
	case "microsoft_ads":
		if w.oauth.MicrosoftClientID == "" || w.oauth.MicrosoftClientSecret == "" {
			return nil
		}
		token, expires, err = provider.RefreshMicrosoftOAuth(ctx, w.httpClient, w.oauth.MicrosoftClientID, w.oauth.MicrosoftClientSecret, *cred)
	case "snapchat":
		if w.oauth.SnapchatClientID == "" || w.oauth.SnapchatClientSecret == "" {
			return nil
		}
		token, newRefresh, expires, err = provider.RefreshSnapchatOAuth(ctx, w.httpClient, w.oauth.SnapchatTokenURL, w.oauth.SnapchatClientID, w.oauth.SnapchatClientSecret, *cred)
	case "linkedin":
		if w.oauth.LinkedInClientID == "" || w.oauth.LinkedInClientSecret == "" {
			return nil
		}
		token, newRefresh, expires, err = provider.RefreshLinkedInOAuth(ctx, w.httpClient, w.oauth.LinkedInTokenURL, w.oauth.LinkedInClientID, w.oauth.LinkedInClientSecret, *cred)
	case "pinterest":
		if w.oauth.PinterestClientID == "" || w.oauth.PinterestClientSecret == "" {
			return nil
		}
		token, newRefresh, expires, err = provider.RefreshPinterestOAuth(ctx, w.httpClient, w.oauth.PinterestTokenURL, w.oauth.PinterestClientID, w.oauth.PinterestClientSecret, *cred)
	case "trafficstars":
		token, expires, err = provider.RefreshTrafficStarsOAuth(ctx, w.httpClient, w.networkBaseURL["trafficstars"], *cred)
	case "mondiad":
		token, newRefresh, expires, err = provider.RefreshMondiadOAuth(ctx, w.httpClient, w.networkBaseURL["mondiad"], *cred)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	cred.AccessToken = token
	cred.ExpiresAt = expires

	enc, err := postback.EncryptAESGCM([]byte(token), w.encryptionKey)
	if err != nil {
		return err
	}
	refreshEnc := row.RefreshTokenEncrypted
	if newRefresh != "" {
		refreshEnc, err = postback.EncryptAESGCM([]byte(newRefresh), w.encryptionKey)
		if err != nil {
			return err
		}
	}
	_, err = db.New(w.pool).UpsertCostSyncCredential(ctx, db.UpsertCostSyncCredentialParams{
		CustomerID:            row.CustomerID,
		Network:               row.Network,
		AccountID:             row.AccountID,
		AccessTokenEncrypted:  enc,
		RefreshTokenEncrypted: refreshEnc,
		ApiKeyEncrypted:       row.ApiKeyEncrypted,
		ExtraConfig:           row.ExtraConfig,
		TokenExpiresAt:        pgtype.Timestamptz{Time: expires, Valid: true},
		SyncIntervalMinutes:   row.SyncIntervalMinutes,
		TokenMapping:          row.TokenMapping,
	})
	return err
}

func (w *Worker) completeRun(ctx context.Context, id int64, status string, rows int, totalUSD int64, errMsg string) {
	var msg pgtype.Text
	if errMsg != "" {
		msg = pgtype.Text{String: errMsg, Valid: true}
	}
	_ = db.New(w.pool).CompleteCostSyncRun(ctx, db.CompleteCostSyncRunParams{
		ID:                  id,
		Status:              status,
		RowsImported:        int32(rows),
		TotalAmountUsdMicro: totalUSD,
		ErrorMessage:        msg,
	})
}

func (w *Worker) tryAdvisoryLock(ctx context.Context) (bool, error) {
	var ok bool
	err := w.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, costSyncAdvisoryLockKey).Scan(&ok)
	return ok, err
}

func (w *Worker) releaseAdvisoryLock(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, advisoryUnlockTimeout)
	defer cancel()
	_, err := w.pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, costSyncAdvisoryLockKey)
	if err == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		slog.Error("cost-sync advisory unlock timed out", "error", err)
		return
	}
	slog.Warn("cost-sync advisory unlock failed", "error", err)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func reconciliationHash(customerID, campaignID uuid.UUID, date time.Time) string {
	raw := fmt.Sprintf("cost_sync_recon|%s|%s|%s", customerID, campaignID, date.Format("2006-01-02"))
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func EncryptCredentialFields(key []byte, accessToken, refreshToken, apiKey string) (accessEnc, refreshEnc, apiEnc []byte, err error) {
	key = normalizeKey(key)
	if accessToken != "" {
		accessEnc, err = postback.EncryptAESGCM([]byte(accessToken), key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if refreshToken != "" {
		refreshEnc, err = postback.EncryptAESGCM([]byte(refreshToken), key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	if apiKey != "" {
		apiEnc, err = postback.EncryptAESGCM([]byte(apiKey), key)
		if err != nil {
			return nil, nil, nil, err
		}
	}
	return accessEnc, refreshEnc, apiEnc, nil
}
