package notify

import (
	"context"

	"github.com/bidshard/ad-event-processor/internal/notify/db"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

type SendNotificationResult struct {
	NotificationID string
	Deduplicated   bool
	Status         db.NotifierNotificationStatus
}

type NotificationInput struct {
	Provider           string
	Recipient          string
	Title              string
	Body               string
	DedupKey           string
	TemplateID         string
	TemplateVars       map[string]string
	AttachmentURL      string
	Broadcast          bool
	BroadcastProviders []string
}

type NotifierAPI interface {
	SendNotification(ctx context.Context, provider, recipient, title, body string) (SendNotificationResult, error)
	SendNotificationInput(ctx context.Context, input NotificationInput) (SendNotificationResult, error)
	SendNotificationBatch(ctx context.Context, inputs []NotificationInput) ([]SendNotificationResult, error)
}

type notifierAPI struct {
	svc *Service
}

func (m *Module) API() NotifierAPI {
	if m == nil || m.svc == nil {
		return nil
	}
	return &notifierAPI{svc: m.svc}
}

func (a *notifierAPI) SendNotification(ctx context.Context, provider, recipient, title, body string) (SendNotificationResult, error) {
	return a.SendNotificationInput(ctx, NotificationInput{
		Provider:  provider,
		Recipient: recipient,
		Title:     title,
		Body:      body,
	})
}

func (a *notifierAPI) SendNotificationInput(ctx context.Context, input NotificationInput) (SendNotificationResult, error) {
	return a.svc.SendNotificationInput(ctx, input)
}

func (a *notifierAPI) SendNotificationBatch(ctx context.Context, inputs []NotificationInput) ([]SendNotificationResult, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	out := make([]SendNotificationResult, 0, len(inputs))
	for _, item := range inputs {
		result, err := a.svc.SendNotificationInput(ctx, item)
		if err != nil {
			return nil, err
		}
		out = append(out, result)
	}
	return out, nil
}
