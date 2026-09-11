package googlesheets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/postback"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool   *pgxpool.Pool
	encKey []byte
}

func NewStore(pool *pgxpool.Pool, encKey []byte) *Store {
	return &Store{pool: pool, encKey: encKey}
}

func (st *Store) OperatorAccountEmail(ctx context.Context, userID uuid.UUID) (string, error) {
	if st == nil || st.pool == nil {
		return "", fmt.Errorf("google sheets store unavailable")
	}
	var email string
	err := st.pool.QueryRow(ctx, `
SELECT u.email
FROM google_sheets_oauth_tokens t
INNER JOIN users u ON u.id = t.user_id
WHERE t.user_id = $1`, userID).Scan(&email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return email, nil
}

func (st *Store) HasConnection(ctx context.Context, userID uuid.UUID) (bool, error) {
	if st == nil || st.pool == nil {
		return false, fmt.Errorf("google sheets store unavailable")
	}
	var exists bool
	err := st.pool.QueryRow(ctx, `
SELECT EXISTS(
  SELECT 1 FROM google_sheets_oauth_tokens WHERE user_id = $1
)`, userID).Scan(&exists)
	return exists, err
}

func (st *Store) UpsertTokens(ctx context.Context, userID uuid.UUID, refreshToken, accessToken string, expiresAt time.Time, scopes string) error {
	if st == nil || st.pool == nil {
		return fmt.Errorf("google sheets store unavailable")
	}
	if len(st.encKey) < 32 {
		return fmt.Errorf("encryption key unavailable")
	}
	refreshEnc, err := postback.EncryptAESGCM([]byte(refreshToken), st.encKey)
	if err != nil {
		return err
	}
	var accessEnc []byte
	if accessToken != "" {
		accessEnc, err = postback.EncryptAESGCM([]byte(accessToken), st.encKey)
		if err != nil {
			return err
		}
	}
	_, err = st.pool.Exec(ctx, `
INSERT INTO google_sheets_oauth_tokens (
  user_id, refresh_token_encrypted, access_token_encrypted, scopes, token_expires_at, updated_at
) VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (user_id) DO UPDATE SET
  refresh_token_encrypted = EXCLUDED.refresh_token_encrypted,
  access_token_encrypted = COALESCE(EXCLUDED.access_token_encrypted, google_sheets_oauth_tokens.access_token_encrypted),
  scopes = EXCLUDED.scopes,
  token_expires_at = EXCLUDED.token_expires_at,
  updated_at = NOW()`, userID, refreshEnc, accessEnc, scopes, nullableTime(expiresAt))
	return err
}

func (st *Store) DeleteConnection(ctx context.Context, userID uuid.UUID) error {
	if st == nil || st.pool == nil {
		return fmt.Errorf("google sheets store unavailable")
	}
	tag, err := st.pool.Exec(ctx, `DELETE FROM google_sheets_oauth_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotConnected
	}
	return nil
}

func (st *Store) LoadTokenRow(ctx context.Context, userID uuid.UUID) (TokenRow, error) {
	if st == nil || st.pool == nil {
		return TokenRow{}, fmt.Errorf("google sheets store unavailable")
	}
	var refreshEnc, accessEnc []byte
	var scopes string
	var expiresAt *time.Time
	err := st.pool.QueryRow(ctx, `
SELECT refresh_token_encrypted, access_token_encrypted, scopes, token_expires_at
FROM google_sheets_oauth_tokens
WHERE user_id = $1`, userID).Scan(&refreshEnc, &accessEnc, &scopes, &expiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TokenRow{}, ErrNotConnected
		}
		return TokenRow{}, err
	}
	refresh, err := postback.DecryptAESGCM(refreshEnc, st.encKey)
	if err != nil {
		return TokenRow{}, err
	}
	out := TokenRow{
		UserID:       userID.String(),
		RefreshToken: string(refresh),
		Scopes:       scopes,
	}
	if len(accessEnc) > 0 {
		access, err := postback.DecryptAESGCM(accessEnc, st.encKey)
		if err != nil {
			return TokenRow{}, err
		}
		out.AccessToken = string(access)
	}
	if expiresAt != nil {
		out.ExpiresAt = expiresAt.UTC()
	}
	return out, nil
}

func nullableTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	utc := t.UTC()
	return &utc
}
