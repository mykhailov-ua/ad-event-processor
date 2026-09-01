import { getDocsSection } from './docs_sections.ts';
import { NAV_ITEMS } from './nav_config.ts';

export type BreadcrumbItem = {
  label: string;
  href?: string;
};

const UUID_RE =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

const STATIC_PATH_LABELS: Record<string, string> = {
  '/customers': 'Customers',
  '/campaigns': 'Campaigns',
  '/billing': 'Billing',
  '/billing/exports': 'Exports',
  '/ops': 'Ops',
  '/ops/dlq': 'DLQ inbox',
  '/ops/blacklist': 'Blacklist',
  '/ops/incidents': 'Incidents',
  '/ops/outbox': 'Outbox',
  '/ops/shards': 'Shards',
  '/ops/ml-model': 'ML model',
  '/ops/domains': 'Domains',
  '/ops/recon': 'Recon',
  '/ops/consent': 'Consent proofs',
  '/ops/rum': 'RUM',
  '/ops/metrics': 'Metrics',
  '/settings': 'Settings',
  '/settings/license': 'License',
  '/settings/licence': 'License',
  '/licence': 'License',
  '/license': 'License',
  '/support/feedback': 'Support feedback',
  '/disputes': 'Disputes',
  '/team': 'Team',
  '/audit': 'Audit',
  '/dashboards': 'Dashboards',
  '/dashboards/campaign': 'Campaign dashboard',
  '/reports': 'Reports',
  '/reports/jobs': 'Report jobs',
  '/rtb': 'RTB',
  '/rtb/deals': 'Deals',
  '/rtb/shadow': 'Shadow',
  '/rtb/floors': 'Floors',
  '/rtb/validate': 'Validate',
  '/rtb/integration-profile': 'Integration',
  '/fraud': 'Fraud',
  '/fraud/integrations': 'Integrations',
  '/fraud/labels': 'ML labels',
  '/fraud/overrides': 'Scoring overrides',
  '/fraud/presets': 'Policy presets',
  '/fraud/decisions': 'Decision explain',
  '/integrations': 'Integrations',
  '/integrations/cost-sync': 'Cost sync',
  '/integrations/postbacks': 'Postbacks',
  '/integrations/schemas': 'Schemas',
  '/integrations/platform-campaigns': 'Platform links',
  '/integrations/affiliate-presets': 'Affiliate presets',
  '/creative': 'Creative',
  '/flows': 'Flows',
  '/landers': 'Landers',
  '/offers': 'Offers',
  '/brands': 'Brands',
  '/supply': 'Supply',
  '/supply/sellers': 'Sellers',
  '/supply/ads-txt': 'ads.txt',
  '/domains': 'Domains',
  '/automation': 'Automation',
  '/automation/presets': 'Automation presets',
  '/automation/rules': 'Automation rules',
  '/traffic-optimizer': 'Traffic optimizer',
  '/traffic-optimizer/presets': 'Optimizer presets',
  '/traffic-optimizer/rules': 'Optimizer rules',
  '/smart-alerts': 'Smart alerts',
  '/smart-alerts/rules': 'Alert rules',
  '/smart-alerts/history': 'Alert history',
  '/margin-guard': 'Margin guard',
  '/margin-guard/policies': 'Margin policies',
  '/margin-guard/activity': 'Margin activity',
  '/portals': 'Portals',
  '/selfserve': 'Self-serve',
  '/publisher': 'Publisher',
  '/publisher/dashboard': 'Publisher dashboard',
  '/publisher/statements': 'Publisher statements',
  '/telegram': 'Telegram',
  '/telegram/bots': 'Telegram bots',
  '/telegram/postbacks': 'Postbacks',
  '/report-schedules': 'Report schedules',
  '/views': 'Saved views',
  '/forecast': 'Forecast',
  '/forecast/campaign': 'Campaign forecast',
  '/docs': 'Documentation',
};

const SEGMENT_LABELS: Record<string, string> = {
  customers: 'Customers',
  campaigns: 'Campaigns',
  billing: 'Billing',
  invoices: 'Invoices',
  exports: 'Exports',
  ops: 'Ops',
  settings: 'Settings',
  license: 'License',
  support: 'Support',
  feedback: 'Feedback',
  disputes: 'Disputes',
  team: 'Team',
  audit: 'Audit',
  dashboards: 'Dashboards',
  campaign: 'Campaign',
  reports: 'Reports',
  jobs: 'Report jobs',
  rtb: 'RTB',
  deals: 'Deals',
  shadow: 'Shadow',
  floors: 'Floors',
  validate: 'Validate',
  fraud: 'Fraud',
  integrations: 'Integrations',
  creative: 'Creative',
  flows: 'Flows',
  landers: 'Landers',
  offers: 'Offers',
  brands: 'Brands',
  supply: 'Supply',
  sellers: 'Sellers',
  domains: 'Domains',
  automation: 'Automation',
  presets: 'Presets',
  rules: 'Rules',
  portals: 'Portals',
  selfserve: 'Self-serve',
  publisher: 'Publisher',
  telegram: 'Telegram',
  bots: 'Telegram bots',
  postbacks: 'Postbacks',
  docs: 'Documentation',
  edit: 'Editor',
  editor: 'Editor',
};

const DASHBOARD_ROLE_LABELS: Record<string, string> = {
  buyer: 'Buyer',
  adops: 'AdOps',
  cfo: 'CFO',
  publisher: 'Publisher',
};

const NON_LINKABLE_PATHS = new Set([
  '/billing/invoices',
  '/dashboards/campaign',
  '/brand-creatives',
]);

for (const item of NAV_ITEMS) {
  STATIC_PATH_LABELS[item.path] = item.label;
}
STATIC_PATH_LABELS['/docs'] = 'Documentation';

type CrumbStep = {
  segment: string;
  path: string;
};

function normalizePathname(pathname: string): string {
  if (!pathname || pathname === '/') {
    return '';
  }
  return pathname.endsWith('/') && pathname.length > 1
    ? pathname.slice(0, -1)
    : pathname;
}

function shortenId(value: string): string {
  if (value.length <= 12) {
    return value;
  }
  return `${value.slice(0, 8)}...`;
}

function humanizeSegment(segment: string): string {
  return segment
    .split('-')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function buildCrumbSteps(segments: string[]): CrumbStep[] {
  if (segments[0] === 'brand-creatives' && segments.length >= 2) {
    const brandId = segments[1] ?? '';
    return [
      { segment: 'brands', path: '/brands' },
      { segment: brandId, path: `/brand-creatives/${brandId}` },
    ];
  }

  return segments.map((segment, index) => ({
    segment,
    path: `/${segments.slice(0, index + 1).join('/')}`,
  }));
}

function resolveStepLabel(
  step: CrumbStep,
  stepIndex: number,
  steps: CrumbStep[],
  segmentLabels: Record<string, string>,
): string {
  const { segment, path } = step;

  if (segmentLabels[segment]) {
    return segmentLabels[segment];
  }

  const staticLabel = STATIC_PATH_LABELS[path];
  if (staticLabel) {
    return staticLabel;
  }

  if (UUID_RE.test(segment)) {
    return shortenId(segment);
  }

  const docsSection = getDocsSection(segment);
  const previousSegment = stepIndex > 0 ? steps[stepIndex - 1]?.segment : undefined;
  if (docsSection && previousSegment === 'docs') {
    return docsSection.title;
  }

  if (previousSegment === 'dashboards' && segment !== 'campaign') {
    return DASHBOARD_ROLE_LABELS[segment] ?? humanizeSegment(segment);
  }

  if (previousSegment === 'reports' && segment !== 'jobs') {
    return humanizeSegment(segment);
  }

  return SEGMENT_LABELS[segment] ?? humanizeSegment(segment);
}

function resolveCrumbHref(steps: CrumbStep[], stepIndex: number): string | undefined {
  if (stepIndex >= steps.length - 1) {
    return undefined;
  }

  const step = steps[stepIndex];
  if (!step) {
    return undefined;
  }

  if (UUID_RE.test(step.segment)) {
    return undefined;
  }

  if (NON_LINKABLE_PATHS.has(step.path)) {
    return undefined;
  }

  return step.path;
}

export function buildBreadcrumbs(
  pathname: string,
  segmentLabels: Record<string, string> = {},
): BreadcrumbItem[] {
  const normalized = normalizePathname(pathname);
  if (!normalized) {
    return [];
  }

  const segments = normalized.split('/').filter(Boolean);
  const steps = buildCrumbSteps(segments);

  return steps.map((step, index) => ({
    label: resolveStepLabel(step, index, steps, segmentLabels),
    href: resolveCrumbHref(steps, index),
  }));
}
