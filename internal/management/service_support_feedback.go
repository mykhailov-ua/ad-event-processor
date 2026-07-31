package management

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"espx/internal/ingestion"
	db "espx/internal/ingestion/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrFeedbackInvalidType  = errors.New("invalid feedback type")
	ErrFeedbackInvalidEmail = errors.New("invalid contact email")
	ErrFeedbackEmptyMessage = errors.New("message is required")
)

type SupportFeedbackInput struct {
	Type          string
	ContactEmail  string
	Message       string
	AttachBundle  bool
	BundleGzip    []byte
	SubmitterID   uuid.UUID
	DeploymentID  string
	BinaryVersion string
	SKU           string
}

type SupportFeedbackMeta struct {
	DeploymentID  string `json:"deployment_id"`
	BinaryVersion string `json:"binary_version"`
	SKU           string `json:"sku"`
}

func (s *Service) SupportFeedbackMeta(ctx context.Context) (SupportFeedbackMeta, error) {
	meta := SupportFeedbackMeta{
		BinaryVersion: os.Getenv("ESPX_BINARY_VERSION"),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if s == nil || s.GetPool() == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	var planCode string
	err := s.GetPool().QueryRow(ctx, `
		SELECT deployment_id, plan_code
		FROM billing.license_status
		LIMIT 1`).Scan(&deploymentID, &planCode)
	if err != nil {
		if err == pgx.ErrNoRows {
			return meta, nil
		}
		return meta, err
	}
	if deploymentID != uuid.Nil {
		meta.DeploymentID = deploymentID.String()
	}
	meta.SKU = planCode
	return meta, nil
}

func (s *Service) RecordSupportFeedback(ctx context.Context, in SupportFeedbackInput) (uuid.UUID, error) {
	feedbackType := strings.ToLower(strings.TrimSpace(in.Type))
	if feedbackType != "bug" && feedbackType != "feature" && feedbackType != "support" {
		return uuid.Nil, ErrFeedbackInvalidType
	}
	email := strings.TrimSpace(in.ContactEmail)
	if email == "" || !strings.Contains(email, "@") {
		return uuid.Nil, ErrFeedbackInvalidEmail
	}
	message := strings.TrimSpace(in.Message)
	if message == "" {
		return uuid.Nil, ErrFeedbackEmptyMessage
	}
	if len(message) > 8000 {
		return uuid.Nil, errValidation("message exceeds 8000 characters")
	}
	if in.AttachBundle && len(in.BundleGzip) == 0 {
		return uuid.Nil, errValidation("bundle attachment required when attach_bundle is true")
	}
	if !in.AttachBundle && len(in.BundleGzip) > 0 {
		in.BundleGzip = nil
	}
	if s == nil || s.GetPool() == nil {
		return uuid.Nil, fmt.Errorf("service unavailable")
	}

	id := uuid.New()
	var submitter pgtype.UUID
	if in.SubmitterID != uuid.Nil {
		submitter = ingestion.ToUUID(in.SubmitterID)
	}
	err := db.New(s.GetPool()).InsertSupportFeedback(ctx, db.InsertSupportFeedbackParams{
		ID:            ingestion.ToUUID(id),
		FeedbackType:  feedbackType,
		ContactEmail:  email,
		Message:       message,
		DeploymentID:  in.DeploymentID,
		BinaryVersion: in.BinaryVersion,
		Sku:           in.SKU,
		AttachBundle:  in.AttachBundle,
		BundleGzip:    in.BundleGzip,
		SubmitterID:   submitter,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert support feedback: %w", err)
	}
	return id, nil
}
