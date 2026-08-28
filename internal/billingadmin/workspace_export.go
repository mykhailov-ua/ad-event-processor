package billingadmin

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

func ParseUsageExportCursor(raw string) (UsageExportCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return UsageExportCursor{}, nil
	}
	parts := strings.SplitN(raw, "|", 3)
	if len(parts) != 3 {
		return UsageExportCursor{}, errInvalidExportCursor("invalid cursor")
	}
	customerID, err := uuid.Parse(parts[0])
	if err != nil {
		return UsageExportCursor{}, errInvalidExportCursor("invalid cursor customer_id")
	}
	usageDate, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		return UsageExportCursor{}, errInvalidExportCursor("invalid cursor usage_date")
	}
	meter := strings.TrimSpace(parts[2])
	if meter == "" {
		return UsageExportCursor{}, errInvalidExportCursor("invalid cursor meter")
	}
	return UsageExportCursor{
		CustomerID: customerID,
		UsageDate:  usageDate,
		Meter:      meter,
		Valid:      true,
	}, nil
}

func (c UsageExportCursor) Encode() string {
	if !c.Valid {
		return ""
	}
	return c.CustomerID.String() + "|" + c.UsageDate.Format("2006-01-02") + "|" + c.Meter
}

func normalizeCostCenter(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) > 64 {
		return "", errValidation("cost_center must be at most 64 characters")
	}
	return trimmed, nil
}

func NormalizeCostCenter(raw string) (string, error) {
	return normalizeCostCenter(raw)
}
