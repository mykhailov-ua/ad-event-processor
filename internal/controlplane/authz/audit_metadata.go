package authz

import (
	"context"

	"github.com/google/uuid"
)

func MergeAuditMetadata(ctx context.Context, metadata any) any {
	user, ok := GetUser(ctx)
	if !ok {
		return metadata
	}
	patch := map[string]any{"auth_source": user.AuthSource}
	if user.AuthSource == "api_key" && user.APIKeyID != uuid.Nil {
		patch["api_key_id"] = user.APIKeyID.String()
	}
	if metadata == nil {
		return patch
	}
	existing, ok := metadata.(map[string]any)
	if !ok {
		wrapped := map[string]any{"payload": metadata, "auth_source": user.AuthSource}
		if user.AuthSource == "api_key" && user.APIKeyID != uuid.Nil {
			wrapped["api_key_id"] = user.APIKeyID.String()
		}
		return wrapped
	}
	out := make(map[string]any, len(existing)+len(patch))
	for key, value := range existing {
		out[key] = value
	}
	for key, value := range patch {
		if value == nil {
			continue
		}
		out[key] = value
	}
	return out
}
