import { api } from './api_client.js';
import { apiConfirmed } from './confirmed_api.js';
import type {
  ApplyCampaignTemplatesResult,
  CreateIntegrationSchemaBody,
  IntegrationSchemaDTO,
  IntegrationTemplateCatalogEntry,
} from '../types/integration.js';
import type { components } from '../types/generated/openapi.js';

export type ApplyIntegrationSchemaResponse =
  components['schemas']['ApplyIntegrationSchemaResponse'];
export type ApplyCampaignTemplatesRequest = components['schemas']['ApplyCampaignTemplatesRequest'];

export const BUNDLED_TRAFFIC_TEMPLATES = [
  { value: 'traffic_propellerads', label: 'PropellerAds' },
  { value: 'traffic_exoclick', label: 'ExoClick' },
  { value: 'traffic_facebook', label: 'Facebook / Meta' },
  { value: 'traffic_popads', label: 'PopAds' },
  { value: 'traffic_trafficstars', label: 'TrafficStars' },
  { value: 'traffic_adsterra', label: 'Adsterra' },
  { value: 'traffic_clickadu', label: 'Clickadu' },
  { value: 'traffic_mgid', label: 'MGID' },
  { value: 'traffic_hilltopads', label: 'HilltopAds' },
  { value: 'traffic_richads', label: 'RichAds' },
  { value: 'traffic_google_ads', label: 'Google Ads' },
  { value: 'traffic_microsoft_ads', label: 'Microsoft Ads' },
  { value: 'traffic_tiktok', label: 'TikTok Ads' },
  { value: 'traffic_taboola', label: 'Taboola' },
  { value: 'traffic_outbrain', label: 'Outbrain' },
  { value: 'traffic_pushground', label: 'Pushground' },
  { value: 'traffic_galaksion', label: 'Galaksion' },
  { value: 'traffic_revcontent', label: 'Revcontent' },
  { value: 'traffic_juicyads', label: 'JuicyAds' },
  { value: 'traffic_evadav', label: 'Evadav' },
  { value: 'traffic_zeropark', label: 'ZeroPark' },
  { value: 'traffic_snapchat', label: 'Snapchat Ads' },
  { value: 'traffic_linkedin', label: 'LinkedIn Ads' },
  { value: 'traffic_instagram', label: 'Instagram Ads' },
  { value: 'traffic_threads', label: 'Threads Ads' },
  { value: 'traffic_pinterest', label: 'Pinterest Ads' },
  { value: 'traffic_x_ads', label: 'X (Twitter) Ads' },
  { value: 'traffic_reddit', label: 'Reddit Ads' },
  { value: 'traffic_youtube', label: 'YouTube Ads' },
  { value: 'traffic_adcash', label: 'Adcash' },
  { value: 'traffic_admaven', label: 'AdMaven' },
  { value: 'traffic_mondiad', label: 'Mondiad' },
  { value: 'traffic_trafficjunky', label: 'TrafficJunky' },
  { value: 'traffic_pushhouse', label: 'Push.House' },
  { value: 'traffic_clickadilla', label: 'ClickAdilla' },
  { value: 'traffic_ezmob', label: 'EZmob' },
  { value: 'traffic_twinred', label: 'TwinRed' },
  { value: 'traffic_trafficfactory', label: 'TrafficFactory' },
  { value: 'traffic_yandex_direct', label: 'Yandex Direct' },
  { value: 'traffic_rollerads', label: 'RollerAds' },
  { value: 'traffic_noviclick', label: 'Noviclick' },
  { value: 'traffic_popcash', label: 'PopCash' },
  { value: 'traffic_adoperator', label: 'AdOperator' },
  { value: 'traffic_traffic_nomads', label: 'Traffic Nomads' },
  { value: 'traffic_admixer', label: 'Admixer' },
  { value: 'traffic_adnium', label: 'Adnium' },
  { value: 'traffic_adsupply', label: 'AdSupply' },
  { value: 'traffic_bitmedia', label: 'Bitmedia' },
  { value: 'traffic_mediago', label: 'MediaGo' },
  { value: 'traffic_rtx_platform', label: 'RTX Platform' },
  { value: 'traffic_sourceknowledge', label: 'SourceKnowledge' },
  { value: 'traffic_reacheffect', label: 'Reacheffect' },
  { value: 'traffic_kadam', label: 'Kadam' },
  { value: 'traffic_plugrush', label: 'PlugRush' },
  { value: 'traffic_runative', label: 'Runative' },
  { value: 'traffic_adskeeper', label: 'AdsKeeper' },
  { value: 'traffic_adxad', label: 'ADxAD' },
  { value: 'traffic_yeesshh', label: 'Yeesshh' },
  { value: 'traffic_adstyle', label: 'Ad.Style' },
  { value: 'traffic_advedro', label: 'Advedro' },
  { value: 'traffic_geospot_media', label: 'GeoSpot Media' },
  { value: 'traffic_rtb_panda', label: 'RTB Panda' },
  { value: 'traffic_widget_media', label: 'Widget Media' },
  { value: 'traffic_onclicka', label: 'OnClickA' },
  { value: 'traffic_bidvertiser', label: 'BidVertiser' },
  { value: 'traffic_dao_ad', label: 'DaoAd' },
  { value: 'traffic_active_revenue', label: 'ActiveRevenue' },
  { value: 'traffic_clickaine', label: 'Clickaine' },
  { value: 'traffic_targeleon', label: 'Targeleon' },
  { value: 'traffic_smartyads', label: 'SmartyAds' },
  { value: 'traffic_traffic_shark', label: 'TrafficShark' },
  { value: 'traffic_adkernel', label: 'Adkernel' },
  { value: 'traffic_reporo', label: 'Reporo' },
  { value: 'traffic_clickflow', label: 'ClickFlow' },
  { value: 'traffic_epom', label: 'Epom' },
  { value: 'traffic_mybid', label: 'MyBid' },
  { value: 'traffic_pushame', label: 'PushAme' },
  { value: 'traffic_vimmy', label: 'Vimmy' },
  { value: 'traffic_traffic_force', label: 'Traffic Force' },
  { value: 'traffic_admeking', label: 'Admeking' },
  { value: 'traffic_clickstar', label: 'ClickStar' },
  { value: 'traffic_selfadvertiser', label: 'SelfAdvertiser' },
] as const;

export const BUNDLED_AFFILIATE_TEMPLATES = [
  { value: 'affiliate_everad', label: 'Everad' },
  { value: 'affiliate_leadbit', label: 'Leadbit' },
  { value: 'affiliate_adcombo', label: 'AdCombo' },
  { value: 'affiliate_lospollos', label: 'LosPollos' },
  { value: 'affiliate_terraleads', label: 'TerraLeads' },
  { value: 'affiliate_dr_cash', label: 'Dr.cash' },
  { value: 'affiliate_cpamatica', label: 'CPAmatica' },
  { value: 'affiliate_mobidea', label: 'Mobidea' },
  { value: 'affiliate_mylead', label: 'MyLead' },
  { value: 'affiliate_maxbounty', label: 'MaxBounty' },
  { value: 'affiliate_clickdealer', label: 'ClickDealer' },
  { value: 'affiliate_lemonads', label: 'Lemonads' },
  { value: 'affiliate_zeydoo', label: 'Zeydoo' },
  { value: 'affiliate_crakrevenue', label: 'CrakRevenue' },
  { value: 'affiliate_advidi', label: 'Advidi' },
  { value: 'affiliate_alfaleads', label: 'Alfaleads' },
  { value: 'affiliate_leadxchange', label: 'LeadXchange' },
  { value: 'affiliate_olimob', label: 'Olimob' },
  { value: 'affiliate_yepads', label: 'YepAds' },
  { value: 'affiliate_creativeclicks', label: 'CreativeClicks' },
  { value: 'affiliate_cpahub', label: 'CPAHUB' },
  { value: 'affiliate_traffic_company', label: 'Traffic Company' },
  { value: 'affiliate_digistore24', label: 'Digistore24' },
  { value: 'affiliate_m4trix', label: 'M4trix' },
  { value: 'affiliate_nova', label: 'Nova' },
  { value: 'affiliate_pwn_games', label: 'PWN Games' },
  { value: 'affiliate_maxweb', label: 'MaxWeb' },
  { value: 'affiliate_bluepartner', label: 'Bluepartner' },
  { value: 'affiliate_franktrax', label: 'Franktrax' },
  { value: 'affiliate_gold_lead', label: 'Gold Lead' },
  { value: 'affiliate_hugeoffers', label: 'HugeOffers' },
  { value: 'affiliate_media500', label: 'Media500' },
  { value: 'affiliate_mobipium', label: 'MOBIPIUM' },
  { value: 'affiliate_direct_affiliate', label: 'Direct Affiliate' },
  { value: 'affiliate_buygoods', label: 'Buygoods' },
  { value: 'affiliate_kelkoo', label: 'Kelkoo' },
  { value: 'affiliate_adultforce', label: 'Adultforce' },
  { value: 'affiliate_gamesvid', label: 'Gamesvid' },
  { value: 'affiliate_datify', label: 'Datify' },
  { value: 'affiliate_traforama', label: 'Traforama' },
  { value: 'affiliate_ytz', label: 'YTZ' },
  { value: 'affiliate_big_bang_ads', label: 'Big Bang Ads' },
  { value: 'affiliate_clickbank', label: 'ClickBank' },
  { value: 'affiliate_tonic_affiliate', label: 'TONIC' },
  { value: 'affiliate_shakes', label: 'Shakes' },
  { value: 'affiliate_aff1', label: 'Aff1' },
  { value: 'affiliate_royal_partners', label: 'Royal Partners' },
  { value: 'affiliate_pinup_partners', label: 'Pin-Up Partners' },
  { value: 'affiliate_1win_partners', label: '1win Partners' },
  { value: 'affiliate_mostbet_partners', label: 'Mostbet Partners' },
  { value: 'affiliate_ogads', label: 'OGAds' },
  { value: 'affiliate_cpalead', label: 'CPAlead' },
  { value: 'affiliate_aivix', label: 'Aivix' },
  { value: 'affiliate_peerfly', label: 'PeerFly' },
  { value: 'affiliate_kma_biz', label: 'KMA.biz' },
  { value: 'affiliate_gasmobi', label: 'Gasmobi' },
  { value: 'affiliate_performcb', label: 'Perform[cb]' },
  { value: 'affiliate_adwork_media', label: 'AdWork Media' },
  { value: 'affiliate_golden_goose', label: 'Golden Goose' },
  { value: 'affiliate_leadgid', label: 'LeadGid' },
  { value: 'affiliate_monetizer', label: 'Monetizer' },
  { value: 'affiliate_nutryst', label: 'Nutryst' },
  { value: 'affiliate_toro_advertising', label: 'Toro Advertising' },
  { value: 'affiliate_smartadv', label: 'SmartAdv' },
  { value: 'affiliate_arrow_media', label: 'Arrow Media' },
  { value: 'affiliate_revenuelab', label: 'RevenueLab' },
  { value: 'affiliate_convertize', label: 'Convertize' },
  { value: 'affiliate_hugoads', label: 'HugoAds' },
  { value: 'affiliate_instal', label: 'Instal' },
  { value: 'affiliate_cpagrip', label: 'CPAGrip' },
  { value: 'affiliate_wiget', label: 'Wiget' },
  { value: 'affiliate_jm_solution', label: 'JM Solution' },
  { value: 'affiliate_adskill', label: 'Adskill' },
  { value: 'affiliate_gotzha', label: 'Gotzha' },
] as const;

export type IntegrationSchemaKind =
  | 'inbound_tokens'
  | 'outbound_postback'
  | 'affiliate_receive_postback'
  | 'status_mapping';

export const INTEGRATION_SCHEMA_STARTERS: Record<IntegrationSchemaKind, Record<string, unknown>> = {
  inbound_tokens: {
    version: 1,
    tokens: [
      { name: 'gclid', query_key: 'gclid', max_len: 256 },
      { name: 'sub1', query_key: 'sub1', max_len: 128 },
    ],
    macros: [{ name: 'campaign_id', source: 'query', key: 'campaign_id', required: true }],
  },
  outbound_postback: {
    version: 1,
    url_template:
      'https://aff.example.com/postback?click_id={click_id}&payout={payout}&status={status}&sub1={sub1}',
    placeholders: ['click_id', 'payout', 'currency', 'status', 'sub1'],
  },
  affiliate_receive_postback: {
    version: 1,
    receive_url_template:
      'https://{tracking_domain}/track?sub1={clickid}&payout={revenue}&status={status}',
    offer_url_suffix: '&clickid={sub1}',
  },
  status_mapping: {
    version: 1,
    status_map: {
      approved: 'conversion',
      rejected: 'rejected',
      pending: 'pending',
    },
  },
};

export async function fetchIntegrationSchemas(): Promise<IntegrationSchemaDTO[]> {
  const { data } = await api<IntegrationSchemaDTO[]>('/api/v1/integration/schemas');
  return data ?? [];
}

export async function fetchIntegrationSchema(schemaId: string): Promise<IntegrationSchemaDTO> {
  const { data } = await api<IntegrationSchemaDTO>(
    `/api/v1/integration/schemas/${encodeURIComponent(schemaId)}`
  );
  return (
    data ?? {
      id: schemaId,
      name: '',
      version: 0,
      kind: '',
      schema: null,
      created_at: '',
      updated_at: '',
    }
  );
}

export async function createIntegrationSchema(
  body: CreateIntegrationSchemaBody
): Promise<IntegrationSchemaDTO> {
  const res = await apiConfirmed<IntegrationSchemaDTO>('/api/v1/integration/schemas', {
    method: 'POST',
    body: JSON.stringify(body),
  });
  return (
    res.data ?? {
      id: '',
      name: body.name,
      version: body.version,
      kind: '',
      schema: body.schema,
      created_at: '',
      updated_at: '',
    }
  );
}

export async function fetchBundledTemplates(): Promise<IntegrationTemplateCatalogEntry[]> {
  const { data } = await api<IntegrationTemplateCatalogEntry[]>('/api/v1/integration/templates');
  return data ?? [];
}

export async function importBundledTemplates(names: string[]): Promise<IntegrationSchemaDTO[]> {
  const res = await apiConfirmed<IntegrationSchemaDTO[]>('/api/v1/integration/templates/import', {
    method: 'POST',
    body: JSON.stringify({ names }),
  });
  return res.data ?? [];
}

export async function applyIntegrationSchema(
  schemaId: string,
  campaignId: string
): Promise<ApplyIntegrationSchemaResponse> {
  const res = await apiConfirmed<ApplyIntegrationSchemaResponse>(
    `/api/v1/integration/schemas/${encodeURIComponent(schemaId)}/apply`,
    {
      method: 'POST',
      body: JSON.stringify({ campaign_id: campaignId }),
    }
  );
  return res.data ?? { status: 'ok', kind: '' };
}

export async function applyCampaignTemplates(
  campaignId: string,
  body: ApplyCampaignTemplatesRequest
): Promise<ApplyCampaignTemplatesResult> {
  const res = await apiConfirmed<ApplyCampaignTemplatesResult>(
    `/api/v1/campaigns/${encodeURIComponent(campaignId)}/apply-templates`,
    {
      method: 'POST',
      body: JSON.stringify(body),
    }
  );
  return res.data ?? { campaign_id: campaignId };
}
