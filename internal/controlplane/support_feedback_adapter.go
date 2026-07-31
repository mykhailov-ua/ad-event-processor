package controlplane

import (
	"context"

	"espx/internal/controlplane/adminapi"

	"github.com/google/uuid"
)

type supportFeedbackAdapter struct {
	svc *Service
}

func (a supportFeedbackAdapter) SupportFeedbackMeta(ctx context.Context) (adminapi.SupportFeedbackMeta, error) {
	meta, err := a.svc.SupportFeedbackMeta(ctx)
	if err != nil {
		return adminapi.SupportFeedbackMeta{}, err
	}
	return adminapi.SupportFeedbackMeta{
		DeploymentID:  meta.DeploymentID,
		BinaryVersion: meta.BinaryVersion,
		SKU:           meta.SKU,
	}, nil
}

func (a supportFeedbackAdapter) RecordSupportFeedback(ctx context.Context, in adminapi.SupportFeedbackRecord) (uuid.UUID, error) {
	return a.svc.RecordSupportFeedback(ctx, SupportFeedbackInput{
		Type:          in.Type,
		ContactEmail:  in.ContactEmail,
		Message:       in.Message,
		AttachBundle:  in.AttachBundle,
		BundleGzip:    in.BundleGzip,
		SubmitterID:   in.SubmitterID,
		DeploymentID:  in.DeploymentID,
		BinaryVersion: in.BinaryVersion,
		SKU:           in.SKU,
	})
}
