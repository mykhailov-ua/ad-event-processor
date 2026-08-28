package migrationsource

import (
	"fmt"
	"strings"
)

const defaultMigrateBudgetMicro = 100_000_000

func DefaultMigrateBudgetMicro() int64 {
	return defaultMigrateBudgetMicro
}

type ExportCampaignShape struct {
	Name                string
	BudgetLimitMicro    int64
	TargetURL           string
	TrafficTemplateID   string
	ClickQueryParams    map[string]string
	IntegrationSchema   string
	IngressCostParam    string
	PostbackURLTemplate string
	Flow                *ExportFlowShape
}

type ExportFlowPathShape struct {
	Weight     int32
	LanderRef  string
	LanderName string
	LanderURL  string
	OfferRef   string
	OfferName  string
	OfferURL   string
}

type ExportFlowShape struct {
	Name  string
	Paths []ExportFlowPathShape
}

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
		Flow:                mappedFlowToExportShape(m.Flow),
	}
}

func mappedFlowToExportShape(flow *MappedFlow) *ExportFlowShape {
	if flow == nil || len(flow.Paths) == 0 {
		return nil
	}
	out := &ExportFlowShape{Name: strings.TrimSpace(flow.Name)}
	if out.Name == "" {
		out.Name = "imported-flow"
	}
	for _, path := range flow.Paths {
		out.Paths = append(out.Paths, ExportFlowPathShape{
			Weight:     path.Weight,
			LanderRef:  path.LanderRef,
			LanderName: path.LanderName,
			LanderURL:  path.LanderURL,
			OfferRef:   path.OfferRef,
			OfferName:  path.OfferName,
			OfferURL:   path.OfferURL,
		})
	}
	return out
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

func ImportIdempotencyKey(batchKey string, index int) string {
	base := strings.TrimSpace(batchKey)
	if base == "" {
		base = "migrate"
	}
	return fmt.Sprintf("%s:%d", base, index)
}
