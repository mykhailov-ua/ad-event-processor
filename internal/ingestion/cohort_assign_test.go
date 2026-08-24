package ingestion

import (
	"testing"

	"ad-event-processor/internal/domain"
)

func TestAssignCohortVariant_Stable(t *testing.T) {
	t.Parallel()
	variants := []domain.CohortVariant{
		{ID: "control", Weight: 50},
		{ID: "treatment", Weight: 50},
	}
	a, _ := AssignCohortVariant("salt", "user-1", variants)
	b, _ := AssignCohortVariant("salt", "user-1", variants)
	if a != b {
		t.Fatalf("expected stable assignment, got %q then %q", a, b)
	}
}
