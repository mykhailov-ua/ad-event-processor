package platformadmin

import (
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestParseAuditListFilter_parsesOptionalFields(t *testing.T) {
	t.Parallel()

	adminID := uuid.New()
	targetID := uuid.New()
	apiKeyID := uuid.New()

	req := httptest.NewRequest(
		"GET",
		"/api/v1/audit?admin_id="+adminID.String()+
			"&target_id="+targetID.String()+
			"&action=PATCH_CAMPAIGN"+
			"&auth_source=api_key"+
			"&api_key_id="+apiKeyID.String(),
		nil,
	)

	got, err := ParseAuditListFilter(req)
	require.NoError(t, err)
	require.Equal(t, adminID, got.AdminID)
	require.Equal(t, targetID, got.TargetID)
	require.Equal(t, "PATCH_CAMPAIGN", got.Action)
	require.Equal(t, "api_key", got.AuthSource)
	require.Equal(t, apiKeyID.String(), got.APIKeyID)
}

func TestParseAuditListFilter_campaignIdAlias(t *testing.T) {
	t.Parallel()

	targetID := uuid.New()
	req := httptest.NewRequest("GET", "/api/v1/audit?campaign_id="+targetID.String(), nil)

	got, err := ParseAuditListFilter(req)
	require.NoError(t, err)
	require.Equal(t, targetID, got.TargetID)
}

func TestParseAuditListFilter_holdoutInvalidAuthSource(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/audit?auth_source=bearer", nil)

	_, err := ParseAuditListFilter(req)
	require.Error(t, err)
}

func TestParseAuditListFilter_holdoutInvalidAdminID(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/api/v1/audit?admin_id=not-a-uuid", nil)

	_, err := ParseAuditListFilter(req)
	require.Error(t, err)
}

func TestAuditFilterSQLParams_emptyFilterUsesInvalidOptionalFields(t *testing.T) {
	t.Parallel()

	params := auditFilterSQLParams(AuditLogFilter{})
	require.False(t, params.AdminID.Valid)
	require.False(t, params.TargetID.Valid)
	require.False(t, params.Action.Valid)
	require.False(t, params.AuthSource.Valid)
	require.False(t, params.ApiKeyID.Valid)
}
