package controlplane

import (
	"context"
	"errors"
	"testing"
)

func TestRecordSupportFeedback_validation(t *testing.T) {
	t.Parallel()
	svc := &Service{}
	ctx := context.Background()

	_, err := svc.RecordSupportFeedback(ctx, SupportFeedbackRecord{
		Type:         "invalid",
		ContactEmail: "ops@example.com",
		Message:      "hello",
	})
	if !errors.Is(err, ErrFeedbackInvalidType) {
		t.Fatalf("type err=%v", err)
	}

	_, err = svc.RecordSupportFeedback(ctx, SupportFeedbackRecord{
		Type:         "bug",
		ContactEmail: "not-an-email",
		Message:      "hello",
	})
	if !errors.Is(err, ErrFeedbackInvalidEmail) {
		t.Fatalf("email err=%v", err)
	}

	_, err = svc.RecordSupportFeedback(ctx, SupportFeedbackRecord{
		Type:         "bug",
		ContactEmail: "ops@example.com",
		Message:      "   ",
	})
	if !errors.Is(err, ErrFeedbackEmptyMessage) {
		t.Fatalf("message err=%v", err)
	}
}
