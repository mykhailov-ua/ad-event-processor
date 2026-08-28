package platformadmin

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/naming"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SupportFeedbackHost interface {
	Pool() *pgxpool.Pool
	ErrValidation(msg string) error
}

func GetSupportFeedbackMeta(ctx context.Context, host SupportFeedbackHost) (SupportFeedbackMeta, error) {
	meta := SupportFeedbackMeta{
		BinaryVersion: os.Getenv(naming.LegacyVendorEnvKey("BINARY_VERSION")),
	}
	if meta.BinaryVersion == "" {
		meta.BinaryVersion = "dev"
	}
	if host == nil || host.Pool() == nil {
		return meta, nil
	}
	var deploymentID uuid.UUID
	err := host.Pool().QueryRow(ctx, `
		SELECT deployment_id
		FROM billing.license_status
		LIMIT 1`).Scan(&deploymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return meta, nil
		}
		return meta, err
	}
	if deploymentID != uuid.Nil {
		meta.DeploymentID = deploymentID.String()
	}
	return meta, nil
}

func RecordSupportFeedback(ctx context.Context, host SupportFeedbackHost, in SupportFeedbackRecord) (uuid.UUID, error) {
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
		return uuid.Nil, host.ErrValidation("message exceeds 8000 characters")
	}
	if in.AttachBundle && len(in.BundleGzip) == 0 {
		return uuid.Nil, host.ErrValidation("bundle attachment required when attach_bundle is true")
	}
	if !in.AttachBundle && len(in.BundleGzip) > 0 {
		in.BundleGzip = nil
	}
	if host == nil || host.Pool() == nil {
		return uuid.Nil, fmt.Errorf("service unavailable")
	}

	id := uuid.New()
	var submitter pgtype.UUID
	if in.SubmitterID != uuid.Nil {
		submitter = domain.ToUUID(in.SubmitterID)
	}
	err := db.New(host.Pool()).InsertSupportFeedback(ctx, db.InsertSupportFeedbackParams{
		ID:            domain.ToUUID(id),
		FeedbackType:  feedbackType,
		ContactEmail:  email,
		Message:       message,
		DeploymentID:  in.DeploymentID,
		BinaryVersion: in.BinaryVersion,
		Sku:           "",
		AttachBundle:  in.AttachBundle,
		BundleGzip:    in.BundleGzip,
		SubmitterID:   submitter,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("insert support feedback: %w", err)
	}
	return id, nil
}
