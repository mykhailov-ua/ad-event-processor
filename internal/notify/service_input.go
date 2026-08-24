package notify

import (
	"context"
	"fmt"

	"ad-event-processor/internal/notify/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func deliveryModeFromInput(input NotificationInput) db.NotifierDeliveryMode {
	if input.Broadcast {
		return db.NotifierDeliveryModeBROADCAST
	}
	return db.NotifierDeliveryModeFALLBACK
}

func (service *Service) resolveNotificationBodyFromInput(ctx context.Context, input NotificationInput) (string, error) {
	if input.TemplateID == "" {
		if input.Body == "" {
			return "", ErrBodyRequired
		}
		return input.Body, nil
	}

	tmpl, err := service.queries.GetTemplate(ctx, input.TemplateID)
	if err != nil {
		return "", fmt.Errorf("load template %s: %w", input.TemplateID, err)
	}

	vars := make(map[string]string, len(input.TemplateVars)+2)
	for k, v := range input.TemplateVars {
		vars[k] = v
	}
	if input.AttachmentURL != "" {
		vars["attachment_url"] = input.AttachmentURL
	}
	if input.Title != "" {
		vars["title"] = input.Title
	}
	body := RenderTemplate(tmpl.Body, vars)
	if body == "" {
		return "", ErrBodyRequired
	}
	return body, nil
}

func (service *Service) createNotificationFromInput(ctx context.Context, input NotificationInput, body string) (db.NotifierNotification, error) {
	provider, err := ParseProviderName(input.Provider)
	if err != nil {
		return db.NotifierNotification{}, err
	}

	broadcastProviders := make([]string, 0, len(input.BroadcastProviders))
	for _, name := range input.BroadcastProviders {
		provider, err := ParseProviderName(name)
		if err != nil {
			return db.NotifierNotification{}, err
		}
		broadcastProviders = append(broadcastProviders, string(provider))
	}

	id, err := uuid.NewV7()
	if err != nil {
		return db.NotifierNotification{}, fmt.Errorf("generate notification id: %w", err)
	}

	notification, err := service.queries.CreateNotification(ctx, db.CreateNotificationParams{
		ID:                 pgtype.UUID{Bytes: id, Valid: true},
		Provider:           provider,
		Recipient:          input.Recipient,
		Title:              pgtypeTextOptional(input.Title),
		Body:               body,
		DeliveryMode:       deliveryModeFromInput(input),
		BroadcastProviders: broadcastProviders,
		DedupKey:           pgtypeTextOptional(input.DedupKey),
		TemplateID:         pgtypeTextOptional(input.TemplateID),
		TemplateVars:       mustMarshalTemplateVars(input.TemplateVars),
		AttachmentUrl:      pgtypeTextOptional(input.AttachmentURL),
	})
	if err != nil {
		return db.NotifierNotification{}, fmt.Errorf("enqueue notification: %w", err)
	}
	return notification, nil
}

func (service *Service) SendNotificationInput(ctx context.Context, input NotificationInput) (SendNotificationResult, error) {
	if input.Recipient == "" {
		return SendNotificationResult{}, ErrRecipientRequired
	}
	body, err := service.resolveNotificationBodyFromInput(ctx, input)
	if err != nil {
		return SendNotificationResult{}, err
	}
	if service.rateLimiter != nil && !service.rateLimiter.allow(input.Recipient) {
		return SendNotificationResult{}, ErrRateLimited
	}

	if input.DedupKey != "" {
		if existing, ok, err := service.findActiveByDedupKey(ctx, input.DedupKey); err != nil {
			return SendNotificationResult{}, err
		} else if ok {
			return SendNotificationResult{
				NotificationID: uuidString(existing.ID),
				Deduplicated:   true,
				Status:         existing.Status,
			}, nil
		}
	}

	notification, err := service.createNotificationFromInput(ctx, input, body)
	if err != nil {
		return SendNotificationResult{}, err
	}

	return SendNotificationResult{
		NotificationID: uuidString(notification.ID),
		Status:         notification.Status,
	}, nil
}

func mustMarshalTemplateVars(vars map[string]string) []byte {
	b, err := marshalTemplateVarsJSON(vars)
	if err != nil {
		return nil
	}
	return b
}
