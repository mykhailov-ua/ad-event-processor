import type { CampaignOnboardingTemplate, CampaignWizardSessionRequest } from '@/api/types';

export type WizardStepId =
  | 'traffic_source'
  | 'integration_template'
  | 'flow_skeleton'
  | 'budget'
  | 'review';

export type TemplateKey = NonNullable<CampaignWizardSessionRequest['template_key']>;

export const WIZARD_STEPS: { id: WizardStepId; label: string }[] = [
  { id: 'traffic_source', label: 'Traffic' },
  { id: 'integration_template', label: 'Integration' },
  { id: 'flow_skeleton', label: 'Flow' },
  { id: 'budget', label: 'Budget' },
  { id: 'review', label: 'Review' },
];

export function formatTimestamp(iso: string | undefined): string {
  if (!iso) {
    return '-';
  }
  const parsed = Date.parse(iso);
  if (!Number.isFinite(parsed)) {
    return iso;
  }
  return new Date(parsed).toLocaleString();
}

export function stepIndex(step: string | undefined): number {
  const index = WIZARD_STEPS.findIndex((row) => row.id === step);
  return index >= 0 ? index : 0;
}

export function defaultTrafficDraft(template: CampaignOnboardingTemplate | undefined) {
  return {
    name: template?.title ?? '',
    traffic_template_id: 'default_rtb',
    click_query_params: JSON.stringify(template?.sample_macros ?? {}, null, 2),
  };
}

export function defaultIntegrationDraft(template: CampaignOnboardingTemplate | undefined) {
  return {
    integration_schema: template?.integration_schema_refs?.[0] ?? '',
    affiliate_network: '',
    tracking_domain: '',
  };
}

export function defaultFlowDraft(template: CampaignOnboardingTemplate | undefined) {
  return {
    flow_name: template?.default_flow?.flow_name ?? '',
    lander_name: template?.default_flow?.lander?.name ?? '',
    lander_url: template?.default_flow?.lander?.url ?? '',
    offer_name: template?.default_flow?.offer?.name ?? '',
    offer_url: template?.default_flow?.offer?.url ?? '',
  };
}

export function defaultBudgetDraft() {
  return {
    budget_usd: '500.00',
    timezone: 'UTC',
    target_countries: 'US',
  };
}
