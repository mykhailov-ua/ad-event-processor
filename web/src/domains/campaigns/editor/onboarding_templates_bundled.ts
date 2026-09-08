import type { CampaignOnboardingTemplate } from '@/api/types';

// Parity with deploy/schemas/onboarding/catalog.v1.yaml and internal/campaign/onboarding/catalog.v1.yaml.
// Bundled onboarding templates for campaign wizard; live API returns the same shape from GET /api/v1/campaigns/onboarding-templates.

export type BundledOnboardingTemplateKey =
  | 'meta_social_funnel'
  | 'popunder_propeller'
  | 'push_house_funnel'
  | 'native_mgid_funnel';

export type BundledOnboardingWizardDefaults = {
  traffic_template_id: string;
  integration_schema: string;
  campaign_name: string;
  budget_limit_micro: number;
  timezone: string;
  target_countries: string[];
};

export const BUNDLED_ONBOARDING_TEMPLATE_KEYS: BundledOnboardingTemplateKey[] = [
  'meta_social_funnel',
  'popunder_propeller',
  'push_house_funnel',
  'native_mgid_funnel',
];

export const BUNDLED_ONBOARDING_TEMPLATES: CampaignOnboardingTemplate[] = [
  {
    key: 'meta_social_funnel',
    title: 'Meta social funnel',
    description: 'Facebook Ads click schema with default lander-to-offer flow.',
    traffic_family: 'social',
    default_flow: {
      flow_name: 'social-main',
      lander: { name: 'Social prelander', url: 'https://example.com/lander' },
      offer: { name: 'Main offer', url: 'https://example.com/offer' },
    },
    integration_schema_refs: ['traffic_facebook'],
    sample_macros: {
      sub2: '{{campaign.id}}',
      ad_campaign_id: '{{campaign.id}}',
      sub3: '{{adset.id}}',
      sub4: '{{ad.id}}',
    },
  },
  {
    key: 'popunder_propeller',
    title: 'PropellerAds popunder',
    description: 'Pop traffic with PropellerAds zone and banner macros.',
    traffic_family: 'pop',
    default_flow: {
      flow_name: 'pop-main',
      lander: { name: 'Pop prelander', url: 'https://example.com/pop-lander' },
      offer: { name: 'Pop offer', url: 'https://example.com/pop-offer' },
    },
    integration_schema_refs: ['traffic_propellerads'],
    sample_macros: {
      sub1: '{zoneid}',
      sub2: '{campaignid}',
      sub3: '{bannerid}',
    },
  },
  {
    key: 'push_house_funnel',
    title: 'Push.house push funnel',
    description: 'Push notification traffic with Push.house site and creative macros.',
    traffic_family: 'push',
    default_flow: {
      flow_name: 'push-main',
      lander: { name: 'Push prelander', url: 'https://example.com/push-lander' },
      offer: { name: 'Push offer', url: 'https://example.com/push-offer' },
    },
    integration_schema_refs: ['traffic_pushhouse'],
    sample_macros: {
      sub1: '{site_id}',
      sub2: '{campaign_id}',
      sub3: '{creative_id}',
    },
  },
  {
    key: 'native_mgid_funnel',
    title: 'MGID native funnel',
    description: 'Native widget traffic with MGID teaser macros.',
    traffic_family: 'native',
    default_flow: {
      flow_name: 'native-main',
      lander: { name: 'Native prelander', url: 'https://example.com/native-lander' },
      offer: { name: 'Native offer', url: 'https://example.com/native-offer' },
    },
    integration_schema_refs: ['traffic_mgid'],
    sample_macros: {
      sub1: '{widget_id}',
      sub2: '{campaign_id}',
      sub3: '{teaser_id}',
    },
  },
];

export const BUNDLED_ONBOARDING_WIZARD_DEFAULTS: Record<
  BundledOnboardingTemplateKey,
  BundledOnboardingWizardDefaults
> = {
  meta_social_funnel: {
    traffic_template_id: 'meta-facebook',
    integration_schema: 'traffic_facebook',
    campaign_name: 'Meta Social Campaign',
    budget_limit_micro: 50_000_000,
    timezone: 'UTC',
    target_countries: ['US'],
  },
  popunder_propeller: {
    traffic_template_id: 'propellerads',
    integration_schema: 'traffic_propellerads',
    campaign_name: 'Propeller Pop Campaign',
    budget_limit_micro: 25_000_000,
    timezone: 'UTC',
    target_countries: ['US', 'CA'],
  },
  push_house_funnel: {
    traffic_template_id: 'push-house',
    integration_schema: 'traffic_pushhouse',
    campaign_name: 'Push House Campaign',
    budget_limit_micro: 30_000_000,
    timezone: 'UTC',
    target_countries: ['US'],
  },
  native_mgid_funnel: {
    traffic_template_id: 'mgid',
    integration_schema: 'traffic_mgid',
    campaign_name: 'MGID Native Campaign',
    budget_limit_micro: 40_000_000,
    timezone: 'UTC',
    target_countries: ['US', 'GB'],
  },
};
