package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"espx/internal/notifier/pb"
)

func RenderTemplate(body string, vars map[string]string) string {
	out := body
	for key, value := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

func (service *Service) RetryNotification(ctx context.Context, notificationID string) (*pb.Notification, error) {
	id, err := pgUUIDFromString(notificationID)
	if err != nil {
		return nil, err
	}
	row, err := service.queries.RetryNotification(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("retry notification: %w", err)
	}
	return notificationToProto(row), nil
}

func marshalTemplateVarsJSON(vars map[string]string) ([]byte, error) {
	if len(vars) == 0 {
		return nil, nil
	}
	return json.Marshal(vars)
}
