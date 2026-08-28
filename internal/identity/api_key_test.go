package identity

import (
	"context"
	"testing"

	"ad-event-processor/internal/identity/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyAPIKey_validSecret(t *testing.T) {
	hasher, err := NewPasswordHasher(32768, 2, 2)
	require.NoError(t, err)

	rawKey := "selfserve-test-key-abcdefghijklmnopqrstuvwxyz"
	keyHash, err := hasher.HashPassword(rawKey)
	require.NoError(t, err)

	userID := uuid.New()
	customerID := uuid.New()
	repo := &verifyAPIKeyMockRepo{
		lookupRow: db.GetAPIKeyByLookupRow{
			ID:         pgtype.UUID{Bytes: uuid.New(), Valid: true},
			UserID:     pgtype.UUID{Bytes: userID, Valid: true},
			KeyHash:    keyHash,
			Role:       "U",
			CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
			Scopes:     []string{"campaigns:read"},
		},
		user: db.User{
			ID:         pgtype.UUID{Bytes: userID, Valid: true},
			Role:       "U",
			CustomerID: pgtype.UUID{Bytes: customerID, Valid: true},
		},
	}

	service := NewService(repo, nil, hasher, nil, nil)
	verified, err := service.VerifyAPIKey(context.Background(), rawKey)
	require.NoError(t, err)
	assert.Equal(t, userID, uuid.UUID(verified.User.ID.Bytes))
	assert.False(t, verified.User.IsBlocked)
	assert.Equal(t, []string{"campaigns:read"}, verified.Scopes)
}

func TestCreateAPIKey_persistsScopes(t *testing.T) {
	repo := &mockRepo{}
	hasher, err := NewPasswordHasher(32768, 2, 2)
	require.NoError(t, err)
	service := NewService(repo, nil, hasher, nil, nil)

	scopes := []string{"campaigns:read", "campaigns:pause"}
	result, err := service.CreateAPIKey(context.Background(), uuid.New(), "scoped", scopes, nil)
	require.NoError(t, err)
	assert.Equal(t, scopes, result.Scopes)
}

func TestVerifyAPIKey_rejectsWrongSecret(t *testing.T) {
	hasher, err := NewPasswordHasher(32768, 2, 2)
	require.NoError(t, err)

	rawKey := "correct-key-value-012345678901234567890"
	keyHash, err := hasher.HashPassword(rawKey)
	require.NoError(t, err)

	repo := &verifyAPIKeyMockRepo{
		lookupRow: db.GetAPIKeyByLookupRow{
			UserID:  pgtype.UUID{Bytes: uuid.New(), Valid: true},
			KeyHash: keyHash,
		},
	}
	service := NewService(repo, nil, hasher, nil, nil)

	_, err = service.VerifyAPIKey(context.Background(), "wrong-key")
	assert.ErrorIs(t, err, ErrInvalidAPIKey)
}

type verifyAPIKeyMockRepo struct {
	mockRepo
	lookupRow db.GetAPIKeyByLookupRow
	user      db.User
}

func (m *verifyAPIKeyMockRepo) GetAPIKeyByLookup(ctx context.Context, keyLookup pgtype.Text) (db.GetAPIKeyByLookupRow, error) {
	if keyLookup.String == apiKeyLookup("wrong-key") {
		return db.GetAPIKeyByLookupRow{}, ErrInvalidAPIKey
	}
	return m.lookupRow, nil
}

func (m *verifyAPIKeyMockRepo) GetUserByID(ctx context.Context, id pgtype.UUID) (db.User, error) {
	return m.user, nil
}
