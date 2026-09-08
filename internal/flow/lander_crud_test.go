package flow

import "testing"

func TestApplyLanderVersionFields_holdout(t *testing.T) {
	dto := &LanderDTO{}
	applyLanderVersionFields(dto, 3, 2)
	if !dto.HasUnpublishedDraft {
		t.Fatal("expected unpublished draft when draft > published")
	}
	applyLanderVersionFields(dto, 2, 2)
	if dto.HasUnpublishedDraft {
		t.Fatal("expected no unpublished draft when versions match")
	}
}
