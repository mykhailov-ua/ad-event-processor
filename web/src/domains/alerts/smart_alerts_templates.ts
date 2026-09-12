import type { SmartAlertRuleTemplate, UpsertSmartAlertRuleRequest } from '@/api/types';

export type { SmartAlertRuleTemplate };

export const SMART_ALERT_TEMPLATE_OPTIONS: Array<{
  value: SmartAlertRuleTemplate;
  label: string;
  hint: string;
}> = [
  {
    value: 'budget_burn_pct',
    label: 'Budget burn %',
    hint: 'Fires when campaign budget burn reaches threshold (percent).',
  },
  {
    value: 'roi_below',
    label: 'ROI below',
    hint: 'Fires when rolling ROI drops below threshold (percent).',
  },
  {
    value: 'pacing_drift',
    label: 'Pacing drift',
    hint: 'Fires when pacing drift exceeds threshold (percent points).',
  },
  {
    value: 'export_job_failed',
    label: 'Export job failed',
    hint: 'Fires when failed export jobs in the last hour reach threshold (count).',
  },
  {
    value: 'margin_breach',
    label: 'Margin breach',
    hint: 'Fires when margin guard activity count reaches threshold.',
  },
];

export function smartAlertTemplateLabel(template: string | undefined): string {
  const match = SMART_ALERT_TEMPLATE_OPTIONS.find((row) => row.value === template);
  return match?.label ?? template ?? 'Custom metric';
}

const TEMPLATE_WIRE_DEFAULTS: Record<
  SmartAlertRuleTemplate,
  Pick<UpsertSmartAlertRuleRequest, 'metric' | 'operator' | 'window_minutes'> & {
    defaultName: string;
  }
> = {
  budget_burn_pct: {
    metric: 'template:budget_burn_pct',
    operator: 'gte',
    window_minutes: 60,
    defaultName: 'Budget burn threshold',
  },
  roi_below: {
    metric: 'template:roi_below',
    operator: 'lt',
    window_minutes: 1440,
    defaultName: 'ROI below threshold',
  },
  pacing_drift: {
    metric: 'template:pacing_drift',
    operator: 'gte',
    window_minutes: 1440,
    defaultName: 'Pacing drift threshold',
  },
  export_job_failed: {
    metric: 'template:export_job_failed',
    operator: 'gte',
    window_minutes: 60,
    defaultName: 'Export job failed',
  },
  margin_breach: {
    metric: 'template:margin_breach',
    operator: 'gte',
    window_minutes: 1440,
    defaultName: 'Margin breach activity',
  },
};

export function buildUpsertSmartAlertRuleFromTemplate(
  template: SmartAlertRuleTemplate,
  fields: {
    customer_id: string;
    threshold: number;
    webhook_url: string;
    enabled: boolean;
    campaign_id?: string;
    name?: string;
  }
): UpsertSmartAlertRuleRequest {
  const spec = TEMPLATE_WIRE_DEFAULTS[template];
  return {
    customer_id: fields.customer_id,
    campaign_id: fields.campaign_id,
    name: fields.name?.trim() || spec.defaultName,
    template,
    metric: spec.metric,
    operator: spec.operator,
    window_minutes: spec.window_minutes,
    threshold: fields.threshold,
    webhook_url: fields.webhook_url,
    enabled: fields.enabled,
  };
}
