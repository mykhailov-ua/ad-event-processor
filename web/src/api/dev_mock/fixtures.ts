import type { Campaign } from '@/api/types';

function devDisplayId(id: string): string {
  let hash = 0;
  for (let index = 0; index < id.length; index += 1) {
    hash = (hash * 31 + id.charCodeAt(index)) >>> 0;
  }
  return String(10_000_000 + (hash % 90_000_000)).padStart(8, '0');
}

export type DevMockCustomer = {
  id: string;
  name: string;
};

export type DevMockUser = {
  id: string;
  email: string;
};

const CUSTOMER_NAMES = [
  'Horizon Media Group',
  'Pacific Ads Studio',
  'Nordic Performance Co',
  'Atlas Buying Desk',
  'Summit Traffiq',
] as const;

const CAMPAIGN_NAMES = [
  'US Summer Surge',
  'EU Retargeting V2',
  'LATAM Mobile App',
  'Global Video Reach',
  'APAC Crypto Swap',
  'Nordic Ecom Promo',
  'DACH High Intent',
  'UK Search Ads Q3',
  'US Gaming Install',
  'SA Ecom Flash',
  'SaaS Leads Global',
  'Fintech Acquisition',
  'B2B Enterprise EU',
  'APAC Direct Sales',
  'US Display Retarget',
  'Crypto Exchange VIP',
  'Mobile Gaming Tier1',
  'EU Ecom Sales',
  'US Performance Push',
  'Global Brand Lift',
  'DE Finance Leads',
  'BR Nutra Push',
  'JP Mobile Subs',
  'CA Insurance CPL',
  'AU Solar Quotes',
] as const;

const STATUSES = ['ACTIVE', 'PAUSED', 'ARCHIVED'] as const;

const COUNTRIES = ['US', 'GB', 'DE', 'CA', 'UA', 'FR', 'JP', 'AU', 'BR', 'MX'] as const;

function devUuid(kind: string, seq: number): string {
  const suffix = String(seq).padStart(12, '0');
  const prefix = kind.slice(0, 4).padEnd(4, '0').slice(0, 4);
  return `00000000-${prefix}-4000-8000-${suffix}`;
}

export const DEV_MOCK_CUSTOMERS: DevMockCustomer[] = CUSTOMER_NAMES.map((name, index) => ({
  id: devUuid('cust', index + 1),
  name,
}));

export const DEV_MOCK_USERS: DevMockUser[] = [
  { id: devUuid('user', 1), email: 'operator@dev.local' },
  { id: devUuid('user', 2), email: 'buyer@dev.local' },
  { id: devUuid('user', 3), email: 'analyst@dev.local' },
];

function campaignStatus(seq: number): (typeof STATUSES)[number] {
  if (seq % 11 === 0) {
    return 'ARCHIVED';
  }
  if (seq % 4 === 0) {
    return 'PAUSED';
  }
  return 'ACTIVE';
}

function buildCampaign(seq: number): Campaign {
  const id = devUuid('camp', seq);
  const customer = DEV_MOCK_CUSTOMERS[(seq - 1) % DEV_MOCK_CUSTOMERS.length];
  const owner = DEV_MOCK_USERS[(seq - 1) % DEV_MOCK_USERS.length];
  const status = campaignStatus(seq);
  const budgetMicro = 5_000_000 + (seq % 7) * 1_250_000;
  const spendMicro = Math.floor(budgetMicro * (0.12 + (seq % 9) * 0.07));
  const now = new Date().toISOString();
  const campaign = {
    id,
    name: CAMPAIGN_NAMES[(seq - 1) % CAMPAIGN_NAMES.length],
    status,
    budget_limit: (budgetMicro / 1_000_000).toFixed(6),
    current_spend: (spendMicro / 1_000_000).toFixed(6),
    customer_id: customer.id,
    pacing_mode: seq % 3 === 0 ? 'ASAP' : 'EVEN',
    daily_budget: (budgetMicro / 30 / 1_000_000).toFixed(6),
    timezone: 'UTC',
    freq_limit: 3,
    freq_window: 86400,
    target_countries: [COUNTRIES[seq % COUNTRIES.length], COUNTRIES[(seq + 3) % COUNTRIES.length]],
    daypart_hours: Array.from({ length: 24 }, (_, hour) => hour),
    owner_user_id: owner.id,
    created_at: now,
    updated_at: now,
    margin_breach: seq % 13 === 0,
  } satisfies Campaign;
  return {
    ...campaign,
    display_id: devDisplayId(id),
  } as Campaign;
}

export function createDevMockCampaigns(): Campaign[] {
  return Array.from({ length: 25 }, (_, index) => buildCampaign(index + 1));
}
