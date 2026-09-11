package reportjob

import (
	"context"
	"testing"
)

func TestSanitizeExportJobError_hidesClickHouse(t *testing.T) {
	got := SanitizeExportJobError("clickhouse: Code: 241. DB::Exception: Memory limit exceeded")
	if got != exportErrSourcePublic {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeExportJobError_timeout(t *testing.T) {
	got := SanitizeExportJobErrorFromErr(context.DeadlineExceeded)
	if got != exportErrTimeoutPublic {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeExportJobError_keepsValidation(t *testing.T) {
	got := SanitizeExportJobError("invalid customer_id")
	if got != "invalid customer_id" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeExportJobError_keepsGoogleSheetsValidation(t *testing.T) {
	got := SanitizeExportJobError("google sheets is not connected for this operator")
	if got != "google sheets is not connected for this operator" {
		t.Fatalf("got %q", got)
	}
}

func TestSanitizeExportJobError_genericInternal(t *testing.T) {
	got := SanitizeExportJobError("pq: unexpected connection reset")
	if got != exportErrGenericPublic {
		t.Fatalf("got %q", got)
	}
}
