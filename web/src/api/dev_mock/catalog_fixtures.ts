import { DEV_MOCK_CUSTOMERS, DEV_MOCK_USERS } from './fixtures.ts';
import { devMockStore } from './store.ts';
import { devMockIso, usdToMicro } from './fixture_helpers.ts';
import {
  emptyParametersSchema,
  SEED_ALERT_RULE_NAMES,
  SEED_AUTOMATION_RULE_NAMES,
  SEED_BRAND_NAMES,
  SEED_CREATIVE_NAMES,
  SEED_DOMAIN_HOSTS,
  SEED_FLOW_NAMES,
  SEED_LANDER_PATHS,
  SEED_MARGIN_POLICY_NAMES,
  SEED_OFFER_NAMES,
  SEED_PUBLISHER_IDS,
  SEED_PUBLISHER_NAMES,
  SEED_SAVED_VIEW_NAMES,
  SEED_TRAFFIC_OPTIMIZER_NAMES,
  seedAdsTxtLine,
  seedAuditTargetId,
  seedCatalogName,
  seedClickId,
  seedDealId,
  seedExternalCampaignId,
  seedTelegramBot,
  seedIpHash,
  seedLanderFileName,
  seedLanderUrl,
  seedPlacementId,
  seedPostbackUrl,
  seedPublisherDomain,
  seedTelegramPostbackUrl,
} from './fixture_names.ts';
import { seedDeterministicUuid } from './seed_uuid.ts';

function campaignId(seq: number): string {
  const campaigns = devMockStore().campaigns;
  return campaigns[(seq - 1) % campaigns.length]?.id ?? seedDeterministicUuid('campaign', seq);
}

function campaignName(seq: number): string {
  const campaigns = devMockStore().campaigns;
  return campaigns[(seq - 1) % campaigns.length]?.name ?? `Campaign ${seq}`;
}

function customerId(seq: number): string {
  return DEV_MOCK_CUSTOMERS[(seq - 1) % DEV_MOCK_CUSTOMERS.length].id;
}

export function devMockCostSyncSnapshot() {
  return {
    networks: [
      { network: 'facebook', display_name: 'Meta Ads', enabled: true },
      { network: 'google', display_name: 'Google Ads', enabled: true },
      { network: 'tiktok', display_name: 'TikTok Ads', enabled: false },
    ],
    credentials: [
      {
        id: seedDeterministicUuid('cost_cred', 1),
        network: 'facebook',
        customer_id: customerId(1),
        sync_interval_minutes: 60,
        last_sync_at: devMockIso(0, 3),
        status: 'ok',
      },
      {
        id: seedDeterministicUuid('cost_cred', 2),
        network: 'google',
        customer_id: customerId(2),
        sync_interval_minutes: 120,
        last_sync_at: devMockIso(1, 5),
        status: 'degraded',
      },
    ],
    history: Array.from({ length: 6 }, (_, index) => ({
      id: seedDeterministicUuid('cost_hist', index + 1),
      network: index % 2 === 0 ? 'facebook' : 'google',
      customer_id: customerId(index + 1),
      started_at: devMockIso(index),
      finished_at: devMockIso(index, 1),
      status: index % 3 === 0 ? 'failed' : 'completed',
      rows_upserted: 120 + index * 40,
    })),
  };
}

export function devMockPostbacksSnapshot() {
  const campaigns = devMockStore().campaigns.slice(0, 6);
  return {
    configs: campaigns.map((campaign, index) => ({
      campaign_id: campaign.id,
      provider: ['webhook', 'facebook', 'google', 'tiktok'][index % 4],
      url_template: seedPostbackUrl(index + 1),
      target_event: 'conversion',
      test_event_code: index % 2 === 0 ? 'TEST_EVENT_42' : '',
      has_api_token: index % 3 !== 0,
    })),
    dlq: Array.from({ length: 4 }, (_, index) => ({
      id: 1000 + index,
      outbox_event_id: 5000 + index,
      campaign_id: campaigns[index % campaigns.length].id,
      click_id: seedClickId(index + 1),
      event_type: 'conversion',
      payload: { goal: 'lead' },
      failures_count: 2 + index,
      last_error: 'upstream timeout',
      status: 'pending',
    })),
    campaignStatus: campaigns.map((campaign, index) => ({
      campaign_id: campaign.id,
      provider: ['webhook', 'facebook'][index % 2],
      last_success_at: devMockIso(index % 5),
      dlq_pending_count: index % 3,
    })),
  };
}

export function devMockIntegrationSnapshot() {
  return {
    schemas: [
      {
        id: seedDeterministicUuid('schema', 1),
        name: 'Affiliate webhook v2',
        version: '2.1',
        updated_at: devMockIso(4),
      },
      {
        id: seedDeterministicUuid('schema', 2),
        name: 'Facebook CAPI',
        version: '1.0',
        updated_at: devMockIso(12),
      },
    ],
    templates: [
      {
        name: 'Affiliate webhook v2',
        file: 'affiliate_webhook_v2.json',
        version: 2,
        category: 'postback',
        kind: 'schema',
      },
      {
        name: 'Facebook purchase',
        file: 'facebook_purchase.json',
        version: 1,
        category: 'capi',
        kind: 'schema',
      },
    ],
  };
}

export function devMockPlatformCampaignLinks() {
  const campaigns = devMockStore().campaigns.slice(0, 10);
  return campaigns.map((campaign, index) => ({
    id: seedDeterministicUuid('platform_link', index + 1),
    campaign_id: campaign.id,
    customer_id: campaign.customer_id,
    network: ['facebook', 'google', 'tiktok', 'taboola'][index % 4],
    external_campaign_id: seedExternalCampaignId(index + 1),
    status: index % 5 === 0 ? 'paused' : 'active',
    last_sync_at: devMockIso(index % 8),
    updated_at: devMockIso(index % 6),
  }));
}

export function devMockBareArrayForPath(pathname: string): unknown[] | undefined {
  const now = devMockIso(0);
  const campaigns = devMockStore().campaigns;

  switch (pathname) {
    case '/api/v1/flows':
      return Array.from({ length: 8 }, (_, index) => ({
        id: seedDeterministicUuid('flow', index + 1),
        name: seedCatalogName(SEED_FLOW_NAMES, index + 1),
        paths: [
          {
            weight: 70,
            lander_id: seedDeterministicUuid('lander', index + 1),
            offer_id: seedDeterministicUuid('offer', index + 1),
          },
        ],
        created_at: devMockIso(index + 3),
      }));
    case '/api/v1/landers':
      return Array.from({ length: 10 }, (_, index) => ({
        id: seedDeterministicUuid('lander', index + 1),
        name: seedLanderFileName(index + 1),
        url: seedLanderUrl(index + 1),
        customer_id: customerId(index + 1),
        created_at: devMockIso(index + 2),
        updated_at: now,
      }));
    case '/api/v1/offers':
      return Array.from({ length: 10 }, (_, index) => ({
        id: seedDeterministicUuid('offer', index + 1),
        name: seedCatalogName(SEED_OFFER_NAMES, index + 1),
        payout_micro: usdToMicro(18 + index * 3.5),
        currency: 'USD',
        customer_id: customerId(index + 1),
        created_at: devMockIso(index + 1),
      }));
    case '/api/v1/brands':
      return DEV_MOCK_CUSTOMERS.map((customer, index) => ({
        id: seedDeterministicUuid('brand', index + 1),
        customer_id: customer.id,
        name: `${seedCatalogName(SEED_BRAND_NAMES, index + 1)} - US East`,
        created_at: devMockIso(index + 5),
        updated_at: now,
        freq_limit: 3,
        freq_window: 86400,
      }));
    case '/api/v1/domains':
      return Array.from({ length: 6 }, (_, index) => ({
        id: seedDeterministicUuid('domain', index + 1),
        hostname: seedCatalogName(SEED_DOMAIN_HOSTS, index + 1),
        customer_id: customerId(index + 1),
        status: index % 4 === 0 ? 'pending' : 'active',
        ssl_status: 'issued',
        created_at: devMockIso(index + 4),
      }));
    case '/api/v1/automation/presets':
      return [
        {
          key: 'pause_on_loss',
          title: 'Pause on loss',
          description: 'Pause when ROI stays below zero for 24 hours.',
          parameters_schema: emptyParametersSchema(),
        },
        {
          key: 'boost_winner_geo',
          title: 'Boost winning GEO',
          description: 'Raise budget on the best-performing country slice.',
          parameters_schema: emptyParametersSchema(),
        },
      ];
    case '/api/v1/automation/rules':
      return Array.from({ length: 5 }, (_, index) => ({
        id: seedDeterministicUuid('automation_rule', index + 1),
        name: seedCatalogName(SEED_AUTOMATION_RULE_NAMES, index + 1),
        enabled: index % 3 !== 0,
        trigger: 'margin_breach',
        customer_id: customerId(index + 1),
        updated_at: devMockIso(index),
      }));
    case '/api/v1/fraud/presets':
      return [
        {
          name: 'strict',
          pass: 20,
          suspect: 45,
          ivt: 70,
          block: 85,
          updated_at: now,
          updated_at_display: now,
        },
        {
          name: 'balanced',
          pass: 35,
          suspect: 60,
          ivt: 80,
          block: 92,
          updated_at: devMockIso(2),
          updated_at_display: devMockIso(2),
        },
      ];
    case '/api/v1/integration/affiliate-status-presets':
      return [
        {
          name: 'Standard affiliate network',
          statuses: [
            { inbound_status: 'approved', goal_name: 'sale' },
            { inbound_status: 'rejected', goal_name: 'lead' },
          ],
        },
        {
          name: 'Lead gen network',
          statuses: [
            { inbound_status: 'confirmed', goal_name: 'lead' },
            { inbound_status: 'hold', goal_name: 'lead' },
          ],
        },
      ];
    case '/api/v1/integration/schemas':
      return devMockIntegrationSnapshot().schemas;
    case '/api/v1/integration/templates':
      return devMockIntegrationSnapshot().templates;
    case '/api/v1/margin-guard/activity':
      return Array.from({ length: 8 }, (_, index) => ({
        id: seedDeterministicUuid('margin_activity', index + 1),
        campaign_id: campaignId(index + 1),
        placement_id: seedPlacementId(index + 1),
        action: index % 2 === 0 ? 'throttle' : 'pause',
        created_at: devMockIso(index),
      }));
    case '/api/v1/margin-guard/policies':
      return Array.from({ length: 4 }, (_, index) => ({
        id: seedDeterministicUuid('margin_policy', index + 1),
        name: seedCatalogName(SEED_MARGIN_POLICY_NAMES, index + 1),
        min_roi_pct: 8 + index * 2,
        customer_id: customerId(index + 1),
        enabled: true,
      }));
    case '/api/v1/postbacks/campaign-status':
      return devMockPostbacksSnapshot().campaignStatus;
    case '/api/v1/postbacks/config':
      return devMockPostbacksSnapshot().configs;
    case '/api/v1/postbacks/dlq':
      return devMockPostbacksSnapshot().dlq;
    case '/api/v1/report-schedules':
      return Array.from({ length: 4 }, (_, index) => ({
        id: seedDeterministicUuid('report_schedule', index + 1),
        customer_id: customerId(index + 1),
        report_key: 'campaign-overview',
        cron: '0 8 * * 1',
        enabled: index % 2 === 0,
      }));
    case '/api/v1/rtb/deals':
      return Array.from({ length: 6 }, (_, index) => ({
        id: index + 1,
        deal_id: seedDealId(index + 1),
        floor_micro: usdToMicro(0.35 + index * 0.08),
        geo_mask: 1 << index % 5,
        cat_mask: 1,
        pacing: index % 2 === 0 ? 'even' : 'asap',
        seats: 2 + index,
        customer_id: customerId(index + 1),
        created_at: devMockIso(index + 6),
        updated_at: now,
      }));
    case '/api/v1/smart-alerts/history':
      return Array.from({ length: 10 }, (_, index) => ({
        id: seedDeterministicUuid('alert_hist', index + 1),
        rule_id: seedDeterministicUuid('alert_rule', (index % 3) + 1),
        campaign_id: campaignId(index + 1),
        severity: index % 3 === 0 ? 'critical' : 'warn',
        message: `Spend velocity spike on ${campaignName(index + 1)}`,
        created_at: devMockIso(index),
      }));
    case '/api/v1/smart-alerts/rules':
      return Array.from({ length: 4 }, (_, index) => ({
        id: seedDeterministicUuid('alert_rule', index + 1),
        name: seedCatalogName(SEED_ALERT_RULE_NAMES, index + 1),
        enabled: true,
        metric: 'spend_velocity',
        customer_id: customerId(index + 1),
      }));
    case '/api/v1/supply/ads-txt':
      return Array.from({ length: 5 }, (_, index) => ({
        id: seedDeterministicUuid('ads_txt', index + 1),
        domain: seedPublisherDomain(index + 1),
        line: seedAdsTxtLine(index + 1),
        updated_at: devMockIso(index),
      }));
    case '/api/v1/supply/sellers':
      return Array.from({ length: 6 }, (_, index) => ({
        id: index + 1,
        seller_id: seedCatalogName(SEED_PUBLISHER_IDS, index + 1),
        name: seedCatalogName(SEED_PUBLISHER_NAMES, index + 1),
        domain: seedPublisherDomain(index + 1),
        seller_type: index % 2 === 0 ? 'PUBLISHER' : 'INTERMEDIARY',
        is_confidential: index % 4 === 0,
        customer_id: customerId(index + 1),
        created_at: devMockIso(index + 2),
        updated_at: now,
      }));
    case '/api/v1/telegram/bots':
      return Array.from({ length: 4 }, (_, index) => {
        const bot = seedTelegramBot(index + 1);
        return {
          id: seedDeterministicUuid('telegram_bot', index + 1),
          name: bot.name,
          campaign_id: campaignId(index + 1),
          username: bot.username,
          status: index % 3 === 0 ? 'paused' : 'active',
        };
      });
    case '/api/v1/traffic-optimizer/presets':
      return [
        {
          key: 'conservative_split',
          title: 'Conservative split',
          description: 'Slow weight shifts with strict minimum data gates.',
          parameters_schema: emptyParametersSchema(),
        },
        {
          key: 'aggressive_scale',
          title: 'Aggressive scale',
          description: 'Faster reallocation toward top paths and sources.',
          parameters_schema: emptyParametersSchema(),
        },
      ];
    case '/api/v1/traffic-optimizer/rules':
      return Array.from({ length: 5 }, (_, index) => ({
        id: seedDeterministicUuid('traffic_opt', index + 1),
        name: seedCatalogName(SEED_TRAFFIC_OPTIMIZER_NAMES, index + 1),
        campaign_id: campaignId(index + 1),
        enabled: index % 2 === 0,
      }));
    case '/api/v1/views':
      return Array.from({ length: 5 }, (_, index) => ({
        id: seedDeterministicUuid('saved_view', index + 1),
        name: seedCatalogName(SEED_SAVED_VIEW_NAMES, index + 1),
        report_key: 'campaign-overview',
        customer_id: customerId(index + 1),
        owner_user_id: DEV_MOCK_USERS[index % DEV_MOCK_USERS.length].id,
      }));
    case '/api/v1/cost-sync/networks':
      return devMockCostSyncSnapshot().networks;
    case '/api/v1/cost-sync/credentials':
      return devMockCostSyncSnapshot().credentials;
    case '/api/v1/cost-sync/history':
      return devMockCostSyncSnapshot().history;
    case '/api/v1/platform-campaigns/links':
      return devMockPlatformCampaignLinks();
    default:
      break;
  }

  if (pathname.startsWith('/api/v1/fraud/integrations')) {
    return campaigns.slice(0, 8).map((campaign, index) => ({
      campaign_id: campaign.id,
      provider: ['ivt-detector', 'manual'][index % 2],
      enabled: index % 4 !== 0,
      updated_at: devMockIso(index),
    }));
  }
  if (pathname.startsWith('/api/v1/fraud/labels')) {
    return Array.from({ length: 12 }, (_, index) => ({
      id: seedDeterministicUuid('fraud_label', index + 1),
      ip_hash: seedIpHash(index + 1),
      label: index % 2 === 0 ? 'block' : 'suspect',
      campaign_id: campaignId(index + 1),
      created_at: devMockIso(index),
    }));
  }
  if (pathname.startsWith('/api/v1/integration/platform-campaigns')) {
    return devMockPlatformCampaignLinks();
  }
  if (pathname.startsWith('/api/v1/telegram/postbacks')) {
    return Array.from({ length: 6 }, (_, index) => ({
      id: seedDeterministicUuid('tg_postback', index + 1),
      campaign_id: campaignId(index + 1),
      postback_url: seedTelegramPostbackUrl(index + 1),
      status: index % 3 === 0 ? 'failed' : 'ok',
      updated_at: devMockIso(index),
    }));
  }
  if (pathname.startsWith('/api/v1/ops/recon')) {
    return Array.from({ length: 6 }, (_, index) => ({
      id: seedDeterministicUuid('recon', index + 1),
      status: index % 2 === 0 ? 'matched' : 'drift',
      diff_micro: index % 2 === 0 ? 0 : usdToMicro(12.5),
      created_at: devMockIso(index),
    }));
  }
  const brandCreatives = /^\/api\/v1\/brands\/([^/]+)\/creatives$/.exec(pathname);
  if (brandCreatives) {
    const brandId = decodeURIComponent(brandCreatives[1]);
    return Array.from({ length: 6 }, (_, index) => ({
      id: seedDeterministicUuid('creative', index + 1),
      brand_id: brandId,
      name: seedCatalogName(SEED_CREATIVE_NAMES, index + 1),
      format: index % 2 === 0 ? 'banner' : 'native',
      width: 300,
      height: 250,
      created_at: devMockIso(index),
    }));
  }

  return undefined;
}
