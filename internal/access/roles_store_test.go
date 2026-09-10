package access

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func testRolesPath(t *testing.T) string {
	dir := filepath.Join("var", "access_test")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	return filepath.Join(dir, "roles_"+t.Name()+".yaml")
}

func seedValidYAML(t *testing.T, path string) {
	data := []byte(`version: 1
revision: 1
roles:
  MB:
    scope: team
    permissions:
      - campaigns:read
      - team:read
`)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}

func TestAccessApply_invalidYAMLDoesNotMutateStore_holdout(t *testing.T) {
	path := testRolesPath(t)
	seedValidYAML(t, path)

	policy := authz.NewStore()
	store := NewRolesStore(path, policy)
	require.NoError(t, store.LoadFromDisk())
	beforeRev := store.Revision()
	beforeRoleCount := len(store.Current().Roles)

	bad := []byte("version: 1\nrevision: 2\nroles:\n  BAD:\n    scope: team\n    permissions:\n      - foo:bar\n")
	_, err := store.ApplyBytes(context.Background(), bad, ApplyOptions{AllowWildcard: true})
	require.Error(t, err)

	require.Equal(t, beforeRev, store.Revision())
	require.Len(t, store.Current().Roles, beforeRoleCount)
	require.False(t, policy.RoleExists("BAD"))
}

func TestAccess_customRole_deniedPostbackWrite_holdout(t *testing.T) {
	path := testRolesPath(t)
	yaml := `version: 1
revision: 1
roles:
  POSTBACK_VIEW:
    scope: team
    permissions:
      - postbacks:read
`
	require.NoError(t, os.WriteFile(path, []byte(yaml), 0o644))

	policy := authz.NewStore()
	store := NewRolesStore(path, policy)
	require.NoError(t, store.LoadFromDisk())

	snap := policy.EffectivePermissions(uuid.Nil, "POSTBACK_VIEW")
	require.True(t, snap.Has("postbacks:read"))
	require.False(t, snap.Has("postbacks:write"))
}

func TestCompileRole_capabilitiesExpand(t *testing.T) {
	perms := CompileRolePermissions(RoleDefinition{
		Capabilities: []string{"page.campaigns", "action.postbacks.view"},
	})
	require.Contains(t, perms, "campaigns:read:masked")
	require.Contains(t, perms, "postbacks:read")
}

func TestValidateDocument_rejectsUnknownPermission(t *testing.T) {
	doc := RolesDocument{
		Version: 1,
		Roles: map[string]RoleDefinition{
			"X": {Scope: "team", Permissions: []string{"foo:bar"}},
		},
	}
	errs, _ := ValidateDocument(doc, false)
	require.NotEmpty(t, errs)
	require.Equal(t, "unknown_permission", errs[0].Code)
}

func TestValidateAssignableTeamRole(t *testing.T) {
	doc := RolesDocument{
		Roles: map[string]RoleDefinition{
			"MB": {Scope: "team"},
			"M":  {Scope: "customer"},
		},
	}
	code, err := ValidateAssignableTeamRole(doc, "MB")
	require.NoError(t, err)
	require.Equal(t, "MB", code)

	_, err = ValidateAssignableTeamRole(doc, "M")
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "team"))
}
