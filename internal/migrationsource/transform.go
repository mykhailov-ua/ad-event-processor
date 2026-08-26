package migrationsource

import (
	"fmt"
	"strings"
)

const defaultMigrateBudgetMicro = 100_000_000

// DefaultMigrateBudgetMicro returns the default campaign budget when a source export omits spend.
func DefaultMigrateBudgetMicro() int64 {
	return defaultMigrateBudgetMicro
}

// ExportCampaignShape mirrors controlplane campaign export fields for migration import.
type ExportCampaignShape struct {
	Name                string
	BudgetLimitMicro    int64
	TargetURL           string
	TrafficTemplateID   string
	ClickQueryParams    map[string]string
	IntegrationSchema   string
	IngressCostParam    string
	PostbackURLTemplate string
}

// MappedToExportShape converts a preview row into an import-ready campaign shape.
func MappedToExportShape(m MappedCampaign, namePrefix string, budgetDefaultMicro int64) ExportCampaignShape {
	name := strings.TrimSpace(m.Name)
	if prefix := strings.TrimSpace(namePrefix); prefix != "" {
		name = strings.TrimSpace(prefix + name)
	}
	budget := m.BudgetLimitMicro
	if budget <= 0 {
		budget = budgetDefaultMicro
	}
	if budget <= 0 {
		budget = defaultMigrateBudgetMicro
	}
	return ExportCampaignShape{
		Name:                name,
		BudgetLimitMicro:    budget,
		TargetURL:           strings.TrimSpace(m.TargetURL),
		TrafficTemplateID:   strings.TrimSpace(m.UITemplateID),
		ClickQueryParams:    cloneStringMap(m.ClickQueryParams),
		IntegrationSchema:   strings.TrimSpace(m.IntegrationSchemaName),
		IngressCostParam:    strings.TrimSpace(m.IngressCostParam),
		PostbackURLTemplate: strings.TrimSpace(m.PostbackURLTemplate),
	}
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// ImportIdempotencyKey builds a per-campaign idempotency key from the batch key and index.
func ImportIdempotencyKey(batchKey string, index int) string {
	base := strings.TrimSpace(batchKey)
	if base == "" {
		base = "migrate"
	}
	return fmt.Sprintf("%s:%d", base, index)
}
