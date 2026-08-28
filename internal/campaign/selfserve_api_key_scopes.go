package campaign

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"
)

var selfServeAPIKeyAllowedScopes = []string{
	"campaigns:read",
	"campaigns:read:masked",
	"campaigns:write",
	"campaigns:pause",
	"customers:read",
}

var selfServeAPIKeyForbiddenScopes = []string{
	"audit:read",
	"ops:write",
	"blacklist:write",
	"shards:read",
	"rtb:write",
}

var selfServeAPIKeyForbiddenReportKeys = map[string]struct{}{
	"fraud-evidence-pack":  {},
	"filter-rejects":       {},
	"layer-desync-summary": {},
}

func ValidateSelfServeAPIKeyScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return []string{"campaigns:read"}, nil
	}
	if len(scopes) > 8 {
		return nil, errValidation("too many scopes")
	}
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return nil, errValidation("empty scope")
		}
		if slices.Contains(selfServeAPIKeyForbiddenScopes, scope) {
			return nil, errValidation(fmt.Sprintf("scope %q not allowed for self-serve keys", scope))
		}
		if !slices.Contains(selfServeAPIKeyAllowedScopes, scope) {
			return nil, errValidation(fmt.Sprintf("unsupported scope %q", scope))
		}
		if _, dup := seen[scope]; dup {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out, nil
}

func SelfServeRouteRequiresScope(routeScope string, keyScopes []string) bool {
	if len(keyScopes) == 0 {
		return true
	}
	return slices.Contains(keyScopes, routeScope)
}

func apiKeyScopesAllowReportKey(reportKey string, scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	_, blocked := selfServeAPIKeyForbiddenReportKeys[reportKey]
	return !blocked
}

func RestrictSnapshotForAPIKeyScopes(base authz.Snapshot, scopes []string) authz.Snapshot {
	if len(scopes) == 0 {
		return base
	}
	allowed := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		allowed[scope] = struct{}{}
	}
	perms := make(map[string]struct{}, len(base.Permissions))
	for perm := range base.Permissions {
		if _, ok := allowed[perm]; ok {
			perms[perm] = struct{}{}
		}
	}
	return authz.Snapshot{
		Permissions: perms,
		Mask:        authz.MaskMasked,
	}
}

func DenyScopedAPIKeyOperatorReport(w http.ResponseWriter, r *http.Request, reportKey string) bool {
	user, ok := authz.GetUser(r.Context())
	if !ok || user.AuthSource != "api_key" || len(user.APIKeyScopes) == 0 {
		return false
	}
	if apiKeyScopesAllowReportKey(reportKey, user.APIKeyScopes) {
		return false
	}
	httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: report not allowed for api key scope")
	return true
}
