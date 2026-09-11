package smartalerts

import (
	"fmt"
	"strings"
)

const templateMetricPrefix = "template:"

const (
	TemplateBudgetBurnPct   = "budget_burn_pct"
	TemplateROIBelow        = "roi_below"
	TemplatePacingDrift     = "pacing_drift"
	TemplateExportJobFailed = "export_job_failed"
	TemplateMarginBreach    = "margin_breach"
)

var validAlertTemplates = map[string]struct{}{
	TemplateBudgetBurnPct:   {},
	TemplateROIBelow:        {},
	TemplatePacingDrift:     {},
	TemplateExportJobFailed: {},
	TemplateMarginBreach:    {},
}

type templateRuleSpec struct {
	Metric        string
	Operator      string
	WindowMinutes int
}

func normalizeAlertTemplate(template string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(template))
	if t == "" {
		return "", fmt.Errorf("template is required")
	}
	if _, ok := validAlertTemplates[t]; !ok {
		return "", fmt.Errorf("unsupported template %q", template)
	}
	return t, nil
}

func templateMetricName(template string) string {
	return templateMetricPrefix + template
}

func parseTemplateFromMetric(metric string) (string, bool) {
	metric = strings.ToLower(strings.TrimSpace(metric))
	if !strings.HasPrefix(metric, templateMetricPrefix) {
		return "", false
	}
	template := strings.TrimPrefix(metric, templateMetricPrefix)
	if _, ok := validAlertTemplates[template]; !ok {
		return "", false
	}
	return template, true
}

func resolveTemplateRuleSpec(template string) (templateRuleSpec, error) {
	template, err := normalizeAlertTemplate(template)
	if err != nil {
		return templateRuleSpec{}, err
	}
	switch template {
	case TemplateBudgetBurnPct:
		return templateRuleSpec{
			Metric:        templateMetricName(template),
			Operator:      "gte",
			WindowMinutes: 60,
		}, nil
	case TemplateROIBelow:
		return templateRuleSpec{
			Metric:        templateMetricName(template),
			Operator:      "lt",
			WindowMinutes: 1440,
		}, nil
	case TemplatePacingDrift:
		return templateRuleSpec{
			Metric:        templateMetricName(template),
			Operator:      "gte",
			WindowMinutes: 1440,
		}, nil
	case TemplateExportJobFailed:
		return templateRuleSpec{
			Metric:        templateMetricName(template),
			Operator:      "gte",
			WindowMinutes: 60,
		}, nil
	case TemplateMarginBreach:
		return templateRuleSpec{
			Metric:        templateMetricName(template),
			Operator:      "gte",
			WindowMinutes: 1440,
		}, nil
	default:
		return templateRuleSpec{}, fmt.Errorf("unsupported template %q", template)
	}
}

func defaultTemplateRuleName(template string) string {
	switch template {
	case TemplateBudgetBurnPct:
		return "Budget burn threshold"
	case TemplateROIBelow:
		return "ROI below threshold"
	case TemplatePacingDrift:
		return "Pacing drift threshold"
	case TemplateExportJobFailed:
		return "Export job failed"
	case TemplateMarginBreach:
		return "Margin breach activity"
	default:
		return template
	}
}

func isTemplateMetric(metric string) bool {
	_, ok := parseTemplateFromMetric(metric)
	return ok
}
