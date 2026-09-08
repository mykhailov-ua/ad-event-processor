import { sha1 } from '@/lib/sha1';
import { seedDeterministicUuid } from '@/lib/uuid';

/** Display strings aligned with cmd/admin/seed_catalog.go and seed_buyer_ch.go. */

export const SEED_BRAND_NAMES = [
  'Velox Checkout',
  'Northstar Finance',
  'Pulse Health',
  'Orbit Travel',
  'Nova SaaS',
  'Harbor Insurance',
  'Kite Mobility',
  'Summit Ecom',
  'Lumen EdTech',
  'Crestline VPN',
] as const;

export const SEED_FLOW_NAMES = [
  'Meta social split',
  'Push subscriber route',
  'Native article path',
  'Search brand lane',
  'Popunder direct',
  'Display prospecting',
  'Remarketing checkout',
  'Affiliate coupon mix',
] as const;

export const SEED_LANDER_PATHS = [
  'lander/checkout-v2',
  'lander/hero-video',
  'lander/compare-table',
  'lander/quiz-funnel',
  'lander/app-install',
  'lander/finance-offer-v3',
  'lander/sweeps-mobile',
  'lander/vsl-17min',
] as const;

export const SEED_OFFER_NAMES = [
  'CreditLine Pro (CPL)',
  'Casino Welcome (CPA)',
  'Travel Booking (CPS)',
  'Finance App (CPI)',
  'Insurance Quote (CPL)',
  'Sweeps Entry (CPL)',
  'Solar panel CPL west',
  'VPN annual plans',
] as const;

export const SEED_PUBLISHER_IDS = [
  'pub_northstar',
  'pub_velocity',
  'pub_summit',
  'pub_harbor',
  'pub_cedar',
  'pub_orbit',
] as const;

export const SEED_PUBLISHER_NAMES = [
  'Northstar Publishing',
  'Velocity Media Network',
  'Summit Content Group',
  'Harborfront Digital',
  'Cedar Lane Media',
  'Orbit Audience Co',
] as const;

export const SEED_TRAFFIC_SOURCES = [
  'facebook',
  'google',
  'tiktok',
  'snapchat',
  'native-ads',
  'taboola',
  'email',
  'affiliate',
] as const;

export const SEED_CREATIVE_NAMES = [
  'Hero carousel',
  'Video pre-roll',
  'Static banner',
  'Native card',
  'Interstitial',
  'Playable unit',
] as const;

export const SEED_DOMAIN_HOSTS = [
  'track.horizon-media.io',
  'track.pacific-ads.studio',
  'track.nordic-performance.co',
  'track.atlas-buying.com',
  'track.summit-traffiq.net',
  'track.bluewave-partners.io',
] as const;

export const SEED_TELEGRAM_BOTS = [
  { name: 'Horizon deals assistant', username: 'horizon_deals_bot' },
  { name: 'Pacific offers desk', username: 'pacific_offers_bot' },
  { name: 'Summit promo alerts', username: 'summit_promo_bot' },
  { name: 'Atlas checkout helper', username: 'atlas_checkout_bot' },
] as const;

export const SEED_AUTOMATION_RULE_NAMES = [
  'Pause on margin breach',
  'Throttle high-IVT placement',
  'Resume after budget refill',
  'Blacklist proxy subnet',
  'Notify on pacing drift',
] as const;

export const SEED_ALERT_RULE_NAMES = [
  'Spend velocity spike',
  'Conversion rate drop',
  'Budget cap proximity',
  'Postback DLQ backlog',
] as const;

export const SEED_SAVED_VIEW_NAMES = [
  'Campaign overview - last 7 days',
  'GEO ROI - EU focus',
  'Fraud breakdown - weekly',
  'Click log - checkout funnel',
  'RTB no-bid audit',
] as const;

export const SEED_MARGIN_POLICY_NAMES = [
  'Portfolio floor 12% ROI',
  'Finance vertical guard',
  'Weekend throttle policy',
  'Tier-1 GEO protection',
] as const;

export const SEED_TRAFFIC_OPTIMIZER_NAMES = [
  'Weighted path optimizer',
  'Bid floor lift on winners',
  'Creative rotation balancer',
  'Source quality rebalancer',
  'Weekend traffic shift',
] as const;

export function seedUserEmail(seq: number): string {
  const domains = [
    'horizon-media.io',
    'pacific-ads.studio',
    'nordic-performance.co',
    'atlas-buying.com',
    'summit-traffiq.net',
  ];
  const localParts = [
    'ops',
    'media.buyer',
    'finance',
    'growth',
    'traffic',
    'analytics',
    'partnerships',
    'dev',
    'campaigns',
    'billing',
  ];
  const domain = domains[seq % domains.length];
  const local = localParts[seq % localParts.length];
  return `${local}+${100 + seq}@${domain}`;
}

export function seedCatalogName<T extends readonly string[]>(items: T, seq: number): string {
  return items[(seq - 1) % items.length];
}

export function seedClickId(seq: number): string {
  return seedDeterministicUuid('click', seq);
}

export function seedPlacementId(seq: number): string {
  return seedCatalogName(SEED_LANDER_PATHS, seq);
}

export function seedIpHash(seq: number): string {
  const digest = sha1(new TextEncoder().encode(`fixture-ip:${seq}`));
  return Array.from(digest, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export function seedExternalCampaignId(seq: number): string {
  const hex = seedDeterministicUuid('external_campaign', seq).replace(/-/g, '');
  const slice = hex.slice(0, 12);
  const numeric = BigInt(`0x${slice}`).toString();
  return numeric.padStart(11, '0').slice(0, 11);
}

export function seedDealId(seq: number): string {
  const labels = ['PMP', 'PG', 'Preferred'];
  const geos = ['US', 'EU', 'APAC'];
  const label = seedCatalogName(labels, seq);
  const geo = seedCatalogName(geos, seq + 1);
  return `${label}-${geo}-display-${String(seq).padStart(2, '0')}`;
}

export function seedPublisherDomain(seq: number): string {
  const slug = seedCatalogName(SEED_PUBLISHER_IDS, seq).replace('pub_', '');
  return `${slug}-media.example`;
}

export function seedAdsTxtLine(seq: number): string {
  const pubId = 1_200_000_000 + seq * 17_431;
  return `google.com, ${pubId}, DIRECT, f08c47fec0942fa0`;
}

export function seedLanderFileName(seq: number): string {
  const path = seedCatalogName(SEED_LANDER_PATHS, seq);
  const leaf = path.split('/').pop() ?? 'index';
  return `${leaf}.html`;
}

export function seedLanderUrl(seq: number): string {
  const host = seedCatalogName(SEED_DOMAIN_HOSTS, seq);
  const path = seedCatalogName(SEED_LANDER_PATHS, seq);
  return `https://${host}/${path}.html`;
}

export function seedTelegramBot(seq: number) {
  return SEED_TELEGRAM_BOTS[(seq - 1) % SEED_TELEGRAM_BOTS.length];
}

export function seedTelegramPostbackUrl(seq: number): string {
  const bot = seedTelegramBot(seq);
  return `https://api.telegram.org/bot/${bot.username}/postback`;
}

export function seedPostbackUrl(seq: number): string {
  const host = seedCatalogName(SEED_DOMAIN_HOSTS, seq);
  return `https://${host}/postback/conversion?click_id={click_id}`;
}

export function seedAuditTargetId(targetType: string, seq: number): string {
  return seedDeterministicUuid(targetType, seq);
}

export function emptyParametersSchema() {
  return [] as { key: string; type: string; description?: string }[];
}
