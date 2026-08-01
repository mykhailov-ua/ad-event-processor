package notify

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func RenderTemplate(body string, vars map[string]string) string {
	out := body
	for key, value := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

func (service *Service) RetryNotification(ctx context.Context, notificationID string) (Notification, error) {
	id, err := pgUUIDFromString(notificationID)
	if err != nil {
		return Notification{}, err
	}
	row, err := service.queries.RetryNotification(ctx, id)
	if err != nil {
		return Notification{}, fmt.Errorf("retry notification: %w", err)
	}
	return notificationFromDB(row), nil
}

func marshalTemplateVarsJSON(vars map[string]string) ([]byte, error) {
	if len(vars) == 0 {
		return nil, nil
	}
	return json.Marshal(vars)
}
