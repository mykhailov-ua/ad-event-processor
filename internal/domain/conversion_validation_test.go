package domain

import "testing"

func TestConversionValidationPending(t *testing.T) {
	if ConversionValidationPending(nil) {
		t.Fatal("nil payload")
	}
	if ConversionValidationPending([]byte(`{"goal_name":"lead"}`)) {
		t.Fatal("missing flag")
	}
	if !ConversionValidationPending([]byte(`{"conversion_validation_pending":true}`)) {
		t.Fatal("expected true")
	}
}
