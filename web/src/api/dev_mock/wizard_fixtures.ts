import type {
  CampaignOnboardingTemplate,
  CampaignWizardCommitResult,
  CampaignWizardSession,
} from '@/api/types';

import { DEV_MOCK_CUSTOMERS } from './fixtures.ts';

const TEMPLATES: CampaignOnboardingTemplate[] = [
  {
    key: 'meta_social_funnel',
    title: 'Meta social funnel',
    description: 'Facebook and Instagram click-to-lander flow with conversion mapping.',
    traffic_family: 'social',
    default_flow: {
      flow_name: 'Meta default',
      lander: { name: 'Social pre-lander', url: 'https://example.com/lander' },
      offer: { name: 'Main offer', url: 'https://example.com/offer' },
    },
    integration_schema_refs: ['binom_v1', 'keitaro_v1'],
    sample_macros: { click_id: '{clickid}', campaign_id: '{campaign_id}' },
  },
  {
    key: 'popunder_propeller',
    title: 'Popunder Propeller',
    description: 'Pop traffic with direct offer routing and spend cap defaults.',
    traffic_family: 'pop',
    default_flow: {
      flow_name: 'Popunder main',
      lander: { name: 'Pop bridge', url: 'https://example.com/pop' },
      offer: { name: 'Pop offer', url: 'https://example.com/pop-offer' },
    },
    integration_schema_refs: ['binom_v1'],
    sample_macros: { zone_id: '{zoneid}' },
  },
  {
    key: 'push_house_funnel',
    title: 'Push house funnel',
    description: 'Push notification source with hold-friendly postback mapping.',
    traffic_family: 'push',
    default_flow: {
      flow_name: 'Push route',
      lander: { name: 'Push lander', url: 'https://example.com/push' },
      offer: { name: 'Push offer', url: 'https://example.com/push-offer' },
    },
    integration_schema_refs: ['keitaro_v1'],
    sample_macros: { push_id: '{pushid}' },
  },
  {
    key: 'native_mgid_funnel',
    title: 'Native MGID funnel',
    description: 'Native placements with teaser URL macros and geo targeting.',
    traffic_family: 'native',
    default_flow: {
      flow_name: 'Native MGID',
      lander: { name: 'Native article', url: 'https://example.com/native' },
      offer: { name: 'Native offer', url: 'https://example.com/native-offer' },
    },
    integration_schema_refs: ['binom_v1', 'keitaro_v1'],
    sample_macros: { teaser_id: '{teaser_id}' },
  },
];

type StoredWizardSession = CampaignWizardSession & {
  template_key: string;
};

const sessions = new Map<string, StoredWizardSession>();

function templateByKey(key: string): CampaignOnboardingTemplate | undefined {
  return TEMPLATES.find((row) => row.key === key);
}

function devUuid(prefix: string): string {
  return `00000000-${prefix}-4000-8000-000000000001`;
}

function defaultSteps(template: CampaignOnboardingTemplate) {
  const integrationSchema = template.integration_schema_refs?.[0] ?? 'binom_v1';
  return {
    traffic_source: {
      name: template.title ?? 'Campaign',
      traffic_template_id: 'default_rtb',
      click_query_params: template.sample_macros ?? {},
    },
    integration_template: {
      integration_schema: integrationSchema,
      affiliate_network: '',
      tracking_domain: '',
    },
    flow_skeleton: {
      flow_name: template.default_flow?.flow_name ?? 'Main flow',
      lander: {
        name: template.default_flow?.lander?.name ?? 'Lander',
        url: template.default_flow?.lander?.url ?? 'https://example.com/lander',
      },
      offer: {
        name: template.default_flow?.offer?.name ?? 'Offer',
        url: template.default_flow?.offer?.url ?? 'https://example.com/offer',
      },
    },
    budget: {
      budget_limit_micro: 500_000_000,
      timezone: 'UTC',
      target_countries: ['US'],
    },
  };
}

function reviewFromSteps(
  steps: NonNullable<CampaignWizardSession['steps']>,
): NonNullable<CampaignWizardSession['review']> {
  return {
    preview: {
      campaign_name: steps.traffic_source?.name,
      traffic_template_id: steps.traffic_source?.traffic_template_id,
      integration_schema: steps.integration_template?.integration_schema,
      flow_name: steps.flow_skeleton?.flow_name,
      budget_limit_micro: steps.budget?.budget_limit_micro,
      target_url: steps.flow_skeleton?.offer?.url,
    },
    warning_slugs: [],
  };
}

export function devMockOnboardingTemplates(): CampaignOnboardingTemplate[] {
  return TEMPLATES;
}

export function devMockWizardSessionGet(sessionId: string): MockWizardResult {
  const session = sessions.get(sessionId);
  if (!session) {
    return { status: 404, body: { error: 'campaign wizard session not found' } };
  }
  return { status: 200, body: session };
}

type MockWizardResult = { status: number; body: unknown };

export function devMockWizardSessionPost(body: Record<string, unknown>): MockWizardResult {
  const action = String(body.action ?? '');

  if (action === 'create') {
    const customerId = String(body.customer_id ?? '');
    const templateKey = String(body.template_key ?? '');
    if (!customerId || !templateKey) {
      return { status: 400, body: { error: 'customer_id and template_key are required' } };
    }
    const template = templateByKey(templateKey);
    if (!template) {
      return { status: 400, body: { error: 'unknown template_key' } };
    }
    const sessionId = devUuid('wizd');
    const now = new Date();
    const expires = new Date(now.getTime() + 24 * 60 * 60 * 1000);
    const steps = defaultSteps(template);
    const session: StoredWizardSession = {
      session_id: sessionId,
      customer_id: customerId,
      current_step: 'traffic_source',
      completed_steps: [],
      steps,
      ready_to_commit: false,
      expires_at: expires.toISOString(),
      updated_at: now.toISOString(),
      template_key: templateKey,
    };
    sessions.set(sessionId, session);
    return { status: 201, body: session };
  }

  const sessionId = String(body.session_id ?? '');
  const session = sessions.get(sessionId);
  if (!session) {
    return { status: 404, body: { error: 'campaign wizard session not found' } };
  }

  if (action === 'update') {
    const step = String(body.step ?? '');
    const payload = (body.payload ?? {}) as Record<string, unknown>;
    const steps = { ...session.steps };
    if (step === 'traffic_source') {
      steps.traffic_source = payload as NonNullable<typeof steps.traffic_source>;
    } else if (step === 'integration_template') {
      steps.integration_template = payload as NonNullable<typeof steps.integration_template>;
    } else if (step === 'flow_skeleton') {
      steps.flow_skeleton = payload as NonNullable<typeof steps.flow_skeleton>;
    } else if (step === 'budget') {
      steps.budget = payload as NonNullable<typeof steps.budget>;
    } else {
      return { status: 400, body: { error: 'unknown wizard step' } };
    }
    const completed = Array.from(new Set([...session.completed_steps, step]));
    const stepOrder = ['traffic_source', 'integration_template', 'flow_skeleton', 'budget'] as const;
    const ready = stepOrder.every((item) => completed.includes(item));
    const nextStep = ready
      ? 'review'
      : (stepOrder.find((item) => !completed.includes(item)) ?? 'review');
    const updated: StoredWizardSession = {
      ...session,
      steps,
      completed_steps: completed,
      current_step: nextStep,
      ready_to_commit: ready,
      review: ready ? reviewFromSteps(steps) : undefined,
      updated_at: new Date().toISOString(),
    };
    sessions.set(sessionId, updated);
    return { status: 200, body: updated };
  }

  if (action === 'commit') {
    if (!session.ready_to_commit) {
      return { status: 409, body: { error: 'campaign wizard session incomplete' } };
    }
    const customer = DEV_MOCK_CUSTOMERS.find((row) => row.id === session.customer_id);
    const result: CampaignWizardCommitResult = {
      campaign: {
        id: devUuid('camp'),
        name: session.steps.traffic_source?.name ?? customer?.name ?? 'New campaign',
      },
      published: Boolean(body.publish),
    };
    sessions.delete(sessionId);
    return { status: 200, body: result };
  }

  return { status: 400, body: { error: 'invalid action' } };
}
