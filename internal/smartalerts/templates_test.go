package smartalerts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveTemplateRuleSpec_budgetBurn(t *testing.T) {
	t.Parallel()
	spec, err := resolveTemplateRuleSpec(TemplateBudgetBurnPct)
	require.NoError(t, err)
	require.Equal(t, templateMetricName(TemplateBudgetBurnPct), spec.Metric)
	require.Equal(t, "gte", spec.Operator)
	require.Equal(t, 60, spec.WindowMinutes)
}

func TestResolveUpsertSmartAlertRule_templatePath(t *testing.T) {
	t.Parallel()
	metric, operator, name, template, window, action, err := resolveUpsertSmartAlertRule(UpsertSmartAlertRuleRequest{
		Template:   TemplateROIBelow,
		Threshold:  12.5,
		WebhookURL: "https://hooks.example.com/alerts",
		Enabled:    true,
	})
	require.NoError(t, err)
	require.Equal(t, TemplateROIBelow, template)
	require.Equal(t, "lt", operator)
	require.Equal(t, templateMetricName(TemplateROIBelow), metric)
	require.Equal(t, "ROI below threshold", name)
	require.Equal(t, 1440, window)
	require.Equal(t, ActionNotify, action)
}

func TestResolveUpsertSmartAlertRule_marginBreachPause_holdout(t *testing.T) {
	t.Parallel()
	_, _, _, template, _, action, err := resolveUpsertSmartAlertRule(UpsertSmartAlertRuleRequest{
		Template:   TemplateMarginBreach,
		Threshold:  1,
		WebhookURL: "https://hooks.example.com/alerts",
		Action:     ActionPauseCampaign,
		Enabled:    true,
	})
	require.NoError(t, err)
	require.Equal(t, TemplateMarginBreach, template)
	require.Equal(t, ActionPauseCampaign, action)
}

func TestValidateAlertAction_nonMarginTemplate_holdout(t *testing.T) {
	t.Parallel()
	require.Error(t, validateAlertAction(TemplateROIBelow, ActionPauseCampaign))
}

func TestParseTemplateFromMetric_holdout(t *testing.T) {
	t.Parallel()
	template, ok := parseTemplateFromMetric("template:export_job_failed")
	require.True(t, ok)
	require.Equal(t, TemplateExportJobFailed, template)
	_, ok = parseTemplateFromMetric("clicks")
	require.False(t, ok)
}
