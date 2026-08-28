package campaign

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/pkg/httpresponse"
)

type CampaignPublishBlockedError struct {
	FieldErrors  map[string]string `json:"field_errors"`
	WarningSlugs []string          `json:"warning_slugs,omitempty"`
}

func (e *CampaignPublishBlockedError) Error() string {
	return ErrCampaignPublishBlocked.Error()
}

func (e *CampaignPublishBlockedError) Is(target error) bool {
	return errors.Is(e, ErrCampaignPublishBlocked)
}

func ParsePublishForceQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes":
		return true
	default:
		return false
	}
}

func CanForceCampaignPublish(ctx context.Context) bool {
	user, ok := authz.GetUser(ctx)
	return ok && authz.NormalizeRole(user.Role) == authz.RoleAdmin
}

func WriteCampaignPublishError(w http.ResponseWriter, err error, writeServiceError func(http.ResponseWriter, error)) {
	var blocked *CampaignPublishBlockedError
	if errors.As(err, &blocked) {
		httpresponse.JSON(w, http.StatusUnprocessableEntity, blocked)
		return
	}
	if writeServiceError != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", err.Error())
}
