package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/reportjob"
)

func writeCampaignImportValidationJSON(ctx context.Context, path string, spec reportjob.ReportJobSpec) error {
	kind := migrationsource.SourceKind(strings.TrimSpace(spec.ImportSourceKind))
	if kind == "" {
		return fmt.Errorf("import_source_kind required")
	}
	payload := []byte(strings.TrimSpace(string(spec.ImportPayload)))
	if len(payload) == 0 {
		return fmt.Errorf("import_payload required")
	}
	if len(payload) > migrationsource.MaxPayloadBytes {
		return fmt.Errorf("import_payload too large")
	}
	result, err := migrationsource.Preview(kind, payload, nil)
	if err != nil {
		return errValidation(err.Error())
	}
	body, err := json.Marshal(result)
	if err != nil {
		return err
	}
	return os.WriteFile(path, body, 0o640)
}
