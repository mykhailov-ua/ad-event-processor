import { templateParamMap, trafficSourceById } from '../models/traffic_source_templates.js';
import { buildTemplatedClickURL } from './traffic_source_url.js';

/** Ordered steps for the first-campaign onboarding wizard. */
export type FirstCampaignWizardStep =
  | 'campaign'
  | 'traffic'
  | 'lander'
  | 'test_click'
  | 'test_postback'
  | 'done';

/** Canonical step order from campaign create through postback smoke test. */
export const FIRST_CAMPAIGN_WIZARD_STEPS: FirstCampaignWizardStep[] = [
  'campaign',
  'traffic',
  'lander',
  'test_click',
  'test_postback',
  'done',
];

const TRAFFIC_SOURCE_TO_BUNDLED: Record<string, string> = {
  'meta-facebook': 'traffic_facebook',
  'meta-instagram': 'traffic_instagram',
  'google-ads': 'traffic_google_ads',
  'google-display': 'traffic_google_ads',
  'youtube-ads': 'traffic_youtube',
  'tiktok-ads': 'traffic_tiktok',
  'snapchat-ads': 'traffic_snapchat',
  'x-ads': 'traffic_x_ads',
  'pinterest-ads': 'traffic_pinterest',
  'linkedin-ads': 'traffic_linkedin',
  'microsoft-ads': 'traffic_microsoft_ads',
  taboola: 'traffic_taboola',
  outbrain: 'traffic_outbrain',
  mgid: 'traffic_mgid',
  revcontent: 'traffic_revcontent',
  propellerads: 'traffic_propellerads',
  'push-house': 'traffic_pushhouse',
  richads: 'traffic_richads',
  'richads-pop': 'traffic_richads',
  adsterra: 'traffic_adsterra',
  exoclick: 'traffic_exoclick',
  trafficjunky: 'traffic_trafficjunky',
  juicyads: 'traffic_juicyads',
  trafficstars: 'traffic_trafficstars',
  hilltopads: 'traffic_hilltopads',
  zeropark: 'traffic_zeropark',
  rollerads: 'traffic_rollerads',
  bidvertiser: 'traffic_bidvertiser',
  popcash: 'traffic_popcash',
  popads: 'traffic_popads',
  clickadu: 'traffic_clickadu',
  evadav: 'traffic_evadav',
};

/**
 * Human-readable label for a wizard step (sidebar / header).
 * @param step - Wizard step id.
 */
export function firstCampaignWizardStepLabel(step: FirstCampaignWizardStep): string {
  switch (step) {
    case 'campaign':
      return 'Campaign';
    case 'traffic':
      return 'Traffic source';
    case 'lander':
      return 'Lander';
    case 'test_click':
      return 'Test click';
    case 'test_postback':
      return 'Test postback';
    case 'done':
      return 'Done';
    default:
      return step;
  }
}

/**
 * Zero-based index of a step in {@link FIRST_CAMPAIGN_WIZARD_STEPS}.
 * @param step - Wizard step id.
 */
export function firstCampaignWizardStepIndex(step: FirstCampaignWizardStep): number {
  return FIRST_CAMPAIGN_WIZARD_STEPS.indexOf(step);
}

/**
 * Next step after the current one, or null when already on done.
 * @param step - Current wizard step.
 */
export function nextFirstCampaignWizardStep(
  step: FirstCampaignWizardStep
): FirstCampaignWizardStep | null {
  const index = firstCampaignWizardStepIndex(step);
  if (index < 0 || index >= FIRST_CAMPAIGN_WIZARD_STEPS.length - 1) return null;
  return FIRST_CAMPAIGN_WIZARD_STEPS[index + 1] ?? null;
}

/**
 * Previous step before the current one, or null on the first step.
 * @param step - Current wizard step.
 */
export function prevFirstCampaignWizardStep(
  step: FirstCampaignWizardStep
): FirstCampaignWizardStep | null {
  const index = firstCampaignWizardStepIndex(step);
  if (index <= 0) return null;
  return FIRST_CAMPAIGN_WIZARD_STEPS[index - 1] ?? null;
}

/**
 * Map UI traffic-source template id to bundled integration schema slug for apply-templates.
 * @param trafficSourceId - Id from {@link TRAFFIC_SOURCE_TEMPLATES}.
 */
export function bundledTrafficTemplateForSource(trafficSourceId: string): string | null {
  const key = String(trafficSourceId || '').trim();
  if (!key) return null;
  return TRAFFIC_SOURCE_TO_BUNDLED[key] ?? null;
}

/**
 * Build click URL for the wizard using the selected traffic-source macro map.
 * @param clickTemplate - Platform click URL template or tracking host.
 * @param campaignId - Created campaign UUID.
 * @param trafficSourceId - Traffic source template id.
 */
export function buildWizardClickURL(
  clickTemplate: string,
  campaignId: string,
  trafficSourceId: string
): string {
  const tpl = trafficSourceById(trafficSourceId);
  const params = tpl ? templateParamMap(tpl) : {};
  return buildTemplatedClickURL(clickTemplate, campaignId, params);
}

export type FirstCampaignBasicsInput = {
  customerId: string;
  templateId: string;
  name: string;
  budgetInput: string;
};

/**
 * Validate campaign step fields before POST /api/v1/selfserve/campaigns.
 * @param input - Form values from the campaign step.
 */
export function validateFirstCampaignBasics(input: FirstCampaignBasicsInput): string | null {
  const customerId = String(input.customerId || '').trim();
  if (!customerId) return 'Customer UUID is required';
  if (!/^[0-9a-f-]{36}$/i.test(customerId)) return 'Customer UUID format is invalid';
  if (!String(input.templateId || '').trim()) return 'Campaign template is required';
  if (!String(input.name || '').trim()) return 'Campaign name is required';
  const budget = String(input.budgetInput || '').trim();
  if (budget && !/^-?\d+(\.\d+)?$/.test(budget)) {
    return 'Budget must be a positive decimal or left blank';
  }
  return null;
}

/**
 * Whether the user can leave the current step without an API call.
 * @param step - Current wizard step.
 * @param campaignId - Created campaign id (empty until campaign step succeeds).
 */
export function canLeaveFirstCampaignWizardStep(
  step: FirstCampaignWizardStep,
  campaignId: string
): boolean {
  if (step === 'campaign') return true;
  return Boolean(campaignId);
}
