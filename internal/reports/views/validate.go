package views

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/reportjob"

	"github.com/google/uuid"
)

const maxSavedViewSpecBytes = 8 * 1024

var allowedSavedViewSpecKeys = map[string]struct{}{
	"from":             {},
	"to":               {},
	"compare":          {},
	"compare_from":     {},
	"compare_to":       {},
	"campaign_id":      {},
	"limit":            {},
	"row_limit":        {},
	"columns":          {},
	"from_offset_days": {},
	"to_offset_days":   {},
	"format":           {},
	"destination":      {},
	"notify":           {},
	"google_sheet":     {},
	"import_payload":   {},
	"kind":             {},
	"entry":            {},
	"billing_format":   {},
	"redact_pii":       {},
}

func allowedSavedViewReportKeys() map[string]struct{} {
	keysFn := liveReportExportKeys
	if keysFn == nil {
		return map[string]struct{}{}
	}
	keys := make(map[string]struct{}, len(keysFn())+4)
	for _, key := range keysFn() {
		keys[key] = struct{}{}
	}
	keys[reportjob.SavedViewReportKeyBilling] = struct{}{}
	keys[reportjob.SavedViewReportKeyAudit] = struct{}{}
	return keys
}

var opsOnlySavedViewReportKeys = map[string]struct{}{
	"filter-rejects":       {},
	"fraud-evidence-pack":  {},
	"layer-desync-summary": {},
}

const savedViewMaxRangeDaysBuyer = 7

func validateSavedViewInputForActor(ctx context.Context, name, reportKey string, spec json.RawMessage) error {
	if err := validateSavedViewInput(name, reportKey, spec); err != nil {
		return err
	}
	return validateSavedViewActorPolicy(ctx, reportKey, spec)
}

// ValidateReportViewForActor validates saved view input for the authenticated actor.
func ValidateReportViewForActor(ctx context.Context, name, reportKey string, spec json.RawMessage) error {
	return validateSavedViewInputForActor(ctx, name, reportKey, spec)
}

func validateSavedViewActorPolicy(ctx context.Context, reportKey string, spec json.RawMessage) error {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return nil
	}
	if snap.Mask == authz.MaskMasked {
		if _, blocked := opsOnlySavedViewReportKeys[reportKey]; blocked {
			return validationErr(fmt.Sprintf("report_key %q not allowed for role", reportKey))
		}
	}
	if err := validateSavedViewCustomerScope(ctx, spec); err != nil {
		return err
	}
	if snap.Mask == authz.MaskMasked {
		return validateSavedViewRangeCap(spec, savedViewMaxRangeDaysBuyer)
	}
	return nil
}

func validateSavedViewCustomerScope(ctx context.Context, spec json.RawMessage) error {
	u, ok := authz.GetUser(ctx)
	if !ok || !u.HasBoundCustomer() {
		return nil
	}
	if len(spec) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(spec, &raw); err != nil {
		return validationErr("spec must be a JSON object")
	}
	campaignRaw, ok := raw["campaign_id"]
	if !ok {
		return nil
	}
	var campaignID string
	if err := json.Unmarshal(campaignRaw, &campaignID); err != nil {
		return validationErr("invalid spec.campaign_id")
	}
	return nil
}

func validateSavedViewRangeCap(spec json.RawMessage, maxDays int) error {
	if len(spec) == 0 {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(spec, &raw); err != nil {
		return validationErr("spec must be a JSON object")
	}
	fromRaw, okFrom := raw["from"]
	toRaw, okTo := raw["to"]
	if !okFrom || !okTo {
		return nil
	}
	var fromStr, toStr string
	if err := json.Unmarshal(fromRaw, &fromStr); err != nil {
		return validationErr("invalid spec.from")
	}
	if err := json.Unmarshal(toRaw, &toStr); err != nil {
		return validationErr("invalid spec.to")
	}
	from, to, err := reportjob.ParseReportRangeFromStrings(fromStr, toStr)
	if err != nil {
		return validationErr(err.Error())
	}
	if to.Sub(from) > time.Duration(maxDays)*24*time.Hour {
		return validationErr(fmt.Sprintf("date range exceeds %d days for role", maxDays))
	}
	return nil
}

func validateSavedViewInput(name, reportKey string, spec json.RawMessage) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return validationErr("name is required")
	}
	if len(name) > 128 {
		return validationErr("name must be at most 128 characters")
	}
	reportKey = strings.TrimSpace(reportKey)
	if reportKey == "" {
		return validationErr("report_key is required")
	}
	if _, ok := allowedSavedViewReportKeys()[reportKey]; !ok {
		return validationErr(fmt.Sprintf("unsupported report_key %q", reportKey))
	}
	return validateSavedViewSpec(spec)
}

func validateSavedViewSpec(spec json.RawMessage) error {
	if len(spec) == 0 {
		return nil
	}
	if len(spec) > maxSavedViewSpecBytes {
		return validationErr(fmt.Sprintf("spec exceeds %d bytes", maxSavedViewSpecBytes))
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(spec, &raw); err != nil {
		return validationErr("spec must be a JSON object")
	}
	for key, value := range raw {
		if _, ok := allowedSavedViewSpecKeys[key]; !ok {
			return validationErr(fmt.Sprintf("unsupported spec key %q", key))
		}
		if err := validateSavedViewSpecValue(key, value); err != nil {
			return err
		}
	}
	if fromRaw, ok := raw["from"]; ok {
		if toRaw, hasTo := raw["to"]; hasTo {
			var fromStr, toStr string
			if err := json.Unmarshal(fromRaw, &fromStr); err != nil {
				return validationErr("invalid spec.from")
			}
			if err := json.Unmarshal(toRaw, &toStr); err != nil {
				return validationErr("invalid spec.to")
			}
			if _, _, err := reportjob.ParseReportRangeFromStrings(fromStr, toStr); err != nil {
				return validationErr(err.Error())
			}
		}
	}
	if compareFromRaw, ok := raw["compare_from"]; ok {
		if compareToRaw, hasCompareTo := raw["compare_to"]; hasCompareTo {
			var compareFrom, compareTo string
			if err := json.Unmarshal(compareFromRaw, &compareFrom); err != nil {
				return validationErr("invalid spec.compare_from")
			}
			if err := json.Unmarshal(compareToRaw, &compareTo); err != nil {
				return validationErr("invalid spec.compare_to")
			}
			if err := reportjob.ValidateReportJobCompareStrings(compareFrom, compareTo); err != nil {
				return validationErr(err.Error())
			}
		}
	}
	return nil
}

func savedViewMaskFromContext(ctx context.Context) authz.MaskLevel {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return authz.MaskMasked
	}
	if snap.Mask == "" {
		return authz.MaskMasked
	}
	return snap.Mask
}

func effectiveSharedViewMask(ownerMask, actorMask authz.MaskLevel) authz.MaskLevel {
	if ownerMask == "" {
		ownerMask = authz.MaskMasked
	}
	if actorMask == authz.MaskMasked || ownerMask == authz.MaskMasked {
		return authz.MaskMasked
	}
	return authz.MaskFull
}

func validateSharedSavedViewForActor(ctx context.Context, view SavedViewDTO) error {
	if !view.IsShared {
		return nil
	}
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return nil
	}
	ownerMask := authz.MaskLevel(view.OwnerMaskLevel)
	effective := effectiveSharedViewMask(ownerMask, snap.Mask)
	policyCtx := authz.WithSnapshot(ctx, authz.Snapshot{
		Permissions: snap.Permissions,
		Mask:        effective,
	})
	if err := validateSavedViewActorPolicy(policyCtx, view.ReportKey, view.Spec); err != nil {
		return ErrForbidden
	}
	return nil
}

func ValidateReportScheduleForActor(ctx context.Context, customerID, reportKey string, spec json.RawMessage) error {
	if err := validateSavedViewInput("schedule", reportKey, spec); err != nil {
		return err
	}
	if err := validateSavedViewActorPolicy(ctx, reportKey, spec); err != nil {
		return err
	}
	u, ok := authz.GetUser(ctx)
	if !ok || !u.HasBoundCustomer() {
		return nil
	}
	customerID = strings.TrimSpace(customerID)
	if customerID == "" {
		return validationErr("customer_id is required")
	}
	if customerID != u.CustomerID.String() {
		return ErrForbidden
	}
	return nil
}

func validateSavedViewSpecValue(key string, value json.RawMessage) error {
	switch key {
	case "from", "to":
		var ts string
		if err := json.Unmarshal(value, &ts); err != nil {
			return validationErr(fmt.Sprintf("invalid spec.%s", key))
		}
		ts = strings.TrimSpace(ts)
		if ts == "" {
			return validationErr(fmt.Sprintf("spec.%s is required", key))
		}
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			return validationErr(fmt.Sprintf("invalid spec.%s timestamp", key))
		}
		return nil
	case "compare":
		var asBool bool
		if err := json.Unmarshal(value, &asBool); err == nil {
			return nil
		}
		var asString string
		if err := json.Unmarshal(value, &asString); err != nil {
			return validationErr("invalid spec.compare")
		}
		if asString != "" && asString != "previous" {
			return validationErr("spec.compare must be previous when set as string")
		}
		return nil
	case "campaign_id":
		var id string
		if err := json.Unmarshal(value, &id); err != nil {
			return validationErr("invalid spec.campaign_id")
		}
		if _, err := uuid.Parse(strings.TrimSpace(id)); err != nil {
			return validationErr("invalid spec.campaign_id")
		}
		return nil
	case "limit", "from_offset_days", "to_offset_days", "row_limit":
		var n int
		if err := json.Unmarshal(value, &n); err != nil {
			return validationErr(fmt.Sprintf("invalid spec.%s", key))
		}
		if n < 0 {
			return validationErr(fmt.Sprintf("spec.%s must be non-negative", key))
		}
		if key == "row_limit" && n > reportjob.ExportRowLimitMax {
			return validationErr(fmt.Sprintf("spec.row_limit exceeds %d", reportjob.ExportRowLimitMax))
		}
		return nil
	case "compare_from", "compare_to":
		var ts string
		if err := json.Unmarshal(value, &ts); err != nil {
			return validationErr(fmt.Sprintf("invalid spec.%s", key))
		}
		ts = strings.TrimSpace(ts)
		if ts == "" {
			return validationErr(fmt.Sprintf("spec.%s is required", key))
		}
		if _, err := time.Parse(time.RFC3339, ts); err != nil {
			return validationErr(fmt.Sprintf("invalid spec.%s timestamp", key))
		}
		return nil
	case "format":
		var format string
		if err := json.Unmarshal(value, &format); err != nil {
			return validationErr("invalid spec.format")
		}
		switch strings.TrimSpace(format) {
		case "csv", "json", "xlsx", "zip", "ndjson":
			return nil
		default:
			return validationErr("unsupported spec.format")
		}
	case "destination":
		var destination string
		if err := json.Unmarshal(value, &destination); err != nil {
			return validationErr("invalid spec.destination")
		}
		switch strings.TrimSpace(destination) {
		case "download", "google_sheet", "":
			return nil
		default:
			return validationErr("unsupported spec.destination")
		}
	case "kind":
		var kind string
		if err := json.Unmarshal(value, &kind); err != nil {
			return validationErr("invalid spec.kind")
		}
		switch strings.TrimSpace(kind) {
		case "report", "billing", "audit":
			return nil
		default:
			return validationErr("unsupported spec.kind")
		}
	case "billing_format":
		var format string
		if err := json.Unmarshal(value, &format); err != nil {
			return validationErr("invalid spec.billing_format")
		}
		switch strings.TrimSpace(format) {
		case "csv", "ndjson":
			return nil
		default:
			return validationErr("unsupported spec.billing_format")
		}
	case "entry":
		var entry string
		if err := json.Unmarshal(value, &entry); err != nil {
			return validationErr("invalid spec.entry")
		}
		if len(strings.TrimSpace(entry)) > 128 {
			return validationErr("spec.entry must be at most 128 characters")
		}
		return nil
	case "redact_pii":
		var enabled bool
		if err := json.Unmarshal(value, &enabled); err != nil {
			return validationErr("invalid spec.redact_pii")
		}
		return nil
	case "notify":
		var notify reportjob.ReportJobNotifySpec
		if err := json.Unmarshal(value, &notify); err != nil {
			return validationErr("invalid spec.notify")
		}
		spec := reportjob.ReportJobSpec{Notify: notify}
		return reportjob.ValidateReportJobNotifySpec(&spec)
	case "google_sheet":
		var sheet reportjob.ReportJobGoogleSheetSpec
		if err := json.Unmarshal(value, &sheet); err != nil {
			return validationErr("invalid spec.google_sheet")
		}
		mode := strings.TrimSpace(sheet.Mode)
		if mode != "" && mode != "create" && mode != "append" {
			return validationErr("unsupported spec.google_sheet.mode")
		}
		if len(sheet.SpreadsheetID) > 128 || len(sheet.SheetTitle) > 128 {
			return validationErr("spec.google_sheet fields exceed 128 characters")
		}
		return nil
	case "import_payload":
		if len(value) > 4096 {
			return validationErr("spec.import_payload exceeds 4096 bytes")
		}
		var payload map[string]json.RawMessage
		if err := json.Unmarshal(value, &payload); err != nil {
			return validationErr("spec.import_payload must be a JSON object")
		}
		return nil
	case "columns":
		var cols []string
		if err := json.Unmarshal(value, &cols); err != nil {
			return validationErr("invalid spec.columns")
		}
		if len(cols) > 64 {
			return validationErr("spec.columns exceeds 64 entries")
		}
		for i := range cols {
			col := strings.TrimSpace(cols[i])
			if col == "" || len(col) > 64 {
				return validationErr("invalid spec.columns entry")
			}
		}
		return nil
	default:
		return nil
	}
}
