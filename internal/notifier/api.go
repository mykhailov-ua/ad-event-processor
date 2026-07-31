package notifier

import (
	"context"
	"strings"

	"espx/internal/notifier/pb"
)

type SendNotificationResult struct {
	NotificationID string
	Deduplicated   bool
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
	h *Handler
}

func (m *Module) API() NotifierAPI {
	if m == nil || m.Handler == nil {
		return nil
	}
	return &notifierAPI{h: m.Handler}
}

func ParseProviderName(name string) (pb.Provider, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return pb.Provider_PROVIDER_UNSPECIFIED, ErrUnsupportedProvider
	}
	upper := strings.ToUpper(trimmed)
	if !strings.HasPrefix(upper, "PROVIDER_") {
		upper = "PROVIDER_" + upper
	}
	if v, ok := pb.Provider_value[upper]; ok {
		return pb.Provider(v), nil
	}
	return pb.Provider_PROVIDER_UNSPECIFIED, ErrUnsupportedProvider
}

func NotificationInputFromPB(req *pb.SendNotificationRequest) NotificationInput {
	if req == nil {
		return NotificationInput{}
	}
	broadcastProviders := make([]string, 0, len(req.BroadcastProviders))
	for _, p := range req.BroadcastProviders {
		broadcastProviders = append(broadcastProviders, p.String())
	}
	return NotificationInput{
		Provider:           req.Provider.String(),
		Recipient:          req.Recipient,
		Title:              req.Title,
		Body:               req.Body,
		DedupKey:           req.DedupKey,
		TemplateID:         req.TemplateId,
		TemplateVars:       req.TemplateVars,
		AttachmentURL:      req.AttachmentUrl,
		Broadcast:          req.DeliveryMode == pb.DeliveryMode_DELIVERY_MODE_BROADCAST,
		BroadcastProviders: broadcastProviders,
	}
}

func (input NotificationInput) toPB() (*pb.SendNotificationRequest, error) {
	provider, err := ParseProviderName(input.Provider)
	if err != nil {
		return nil, err
	}
	req := &pb.SendNotificationRequest{
		Provider:      provider,
		Recipient:     input.Recipient,
		Title:         input.Title,
		Body:          input.Body,
		DedupKey:      input.DedupKey,
		TemplateId:    input.TemplateID,
		TemplateVars:  input.TemplateVars,
		AttachmentUrl: input.AttachmentURL,
	}
	if input.Broadcast {
		req.DeliveryMode = pb.DeliveryMode_DELIVERY_MODE_BROADCAST
	}
	if len(input.BroadcastProviders) > 0 {
		req.BroadcastProviders = make([]pb.Provider, 0, len(input.BroadcastProviders))
		for _, name := range input.BroadcastProviders {
			p, err := ParseProviderName(name)
			if err != nil {
				return nil, err
			}
			req.BroadcastProviders = append(req.BroadcastProviders, p)
		}
	}
	return req, nil
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
	req, err := input.toPB()
	if err != nil {
		return SendNotificationResult{}, err
	}
	resp, err := a.h.SendNotification(ctx, req)
	if err != nil {
		return SendNotificationResult{}, err
	}
	return SendNotificationResult{
		NotificationID: resp.NotificationId,
		Deduplicated:   resp.Deduplicated,
	}, nil
}

func (a *notifierAPI) SendNotificationBatch(ctx context.Context, inputs []NotificationInput) ([]SendNotificationResult, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	reqs := make([]*pb.SendNotificationRequest, 0, len(inputs))
	for _, item := range inputs {
		req, err := item.toPB()
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	resp, err := a.h.SendNotificationBatch(ctx, &pb.SendNotificationBatchRequest{Notifications: reqs})
	if err != nil {
		return nil, err
	}
	out := make([]SendNotificationResult, 0, len(resp.Notifications))
	for _, item := range resp.Notifications {
		out = append(out, SendNotificationResult{
			NotificationID: item.NotificationId,
			Deduplicated:   item.Deduplicated,
		})
	}
	return out, nil
}
