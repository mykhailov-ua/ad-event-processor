/**
 * Static traffic-source presets for campaign Integration (click URL macros).
 * Catalog ships in the admin bundle — no DB seed required (§11.3).
 */

export type TrafficSourceCategory =
  | 'social'
  | 'search'
  | 'native'
  | 'push'
  | 'adult'
  | 'dsp'
  | 'direct';

export type CostSyncNetwork = 'meta' | 'google' | 'tiktok';

export type TrafficSourceParam = {
  /** BidShard query key (sub1…sub30, ad_campaign_id, gclid, …). */
  key: string;
  /** Network token or static label — left raw in the click URL when macro-shaped. */
  value: string;
  label?: string;
};

export type TrafficSourceTemplate = {
  id: string;
  name: string;
  category: TrafficSourceCategory;
  /** Cost Sync join network when ad_campaign_id / sub2 carries the external campaign id. */
  cost_sync?: CostSyncNetwork;
  params: TrafficSourceParam[];
  notes?: string;
};

/**
 * True when value is a network macro that must stay unencoded in the pasted URL.
 */
export function isNetworkMacro(value: string): boolean {
  const v = String(value || '');
  return (
    /\{\{[^}]+\}\}/.test(v)
    || /\{[a-zA-Z_][a-zA-Z0-9_.]*\}/.test(v)
    || /__[A-Z0-9_]+__/.test(v)
    || /##[A-Z0-9_]+##/.test(v)
    || /^\[[^\]]+\]$/.test(v)
  );
}

/**
 * Flatten template params into a query map (subN + attribution keys).
 */
export function templateParamMap(tpl: TrafficSourceTemplate): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of tpl.params) {
    if (!p.key) continue;
    out[p.key] = p.value;
  }
  return out;
}

function p(key: string, value: string, label?: string): TrafficSourceParam {
  return label ? { key, value, label } : { key, value };
}

/**
 * Canonical 33-preset catalog (Facebook → Direct).
 */
export const TRAFFIC_SOURCE_TEMPLATES: TrafficSourceTemplate[] = [
  {
    id: 'meta-facebook',
    name: 'Facebook Ads',
    category: 'social',
    cost_sync: 'meta',
    notes: 'Cost Sync join: sub2 / ad_campaign_id = {{campaign.id}}',
    params: [
      p('sub1', '{{site_source_name}}', 'Placement'),
      p('sub2', '{{campaign.id}}', 'Campaign'),
      p('ad_campaign_id', '{{campaign.id}}', 'Cost Sync'),
      p('sub3', '{{adset.id}}', 'Ad set'),
      p('sub4', '{{ad.id}}', 'Ad'),
      p('fbclid', '{{fbclid}}'),
    ],
  },
  {
    id: 'meta-instagram',
    name: 'Instagram Ads',
    category: 'social',
    cost_sync: 'meta',
    params: [
      p('sub1', 'instagram'),
      p('sub2', '{{campaign.id}}'),
      p('ad_campaign_id', '{{campaign.id}}'),
      p('sub3', '{{adset.id}}'),
      p('sub4', '{{ad.id}}'),
      p('fbclid', '{{fbclid}}'),
    ],
  },
  {
    id: 'google-ads',
    name: 'Google Ads',
    category: 'search',
    cost_sync: 'google',
    params: [
      p('sub1', '{network}'),
      p('sub2', '{campaignid}'),
      p('ad_campaign_id', '{campaignid}'),
      p('sub3', '{adgroupid}'),
      p('sub4', '{creative}'),
      p('sub5', '{keyword}'),
      p('gclid', '{gclid}'),
    ],
  },
  {
    id: 'google-display',
    name: 'Google Display',
    category: 'native',
    cost_sync: 'google',
    params: [
      p('sub1', 'gdn'),
      p('sub2', '{campaignid}'),
      p('ad_campaign_id', '{campaignid}'),
      p('sub3', '{placement}'),
      p('gclid', '{gclid}'),
    ],
  },
  {
    id: 'youtube-ads',
    name: 'YouTube Ads',
    category: 'social',
    cost_sync: 'google',
    params: [
      p('sub1', 'youtube'),
      p('sub2', '{campaignid}'),
      p('ad_campaign_id', '{campaignid}'),
      p('sub3', '{adgroupid}'),
      p('gclid', '{gclid}'),
    ],
  },
  {
    id: 'tiktok-ads',
    name: 'TikTok Ads',
    category: 'social',
    cost_sync: 'tiktok',
    params: [
      p('sub1', '__PLACEMENT__'),
      p('sub2', '__CAMPAIGN_ID__'),
      p('ad_campaign_id', '__CAMPAIGN_ID__'),
      p('sub3', '__AID__'),
      p('sub4', '__CID__'),
      p('ttclid', '__CLICKID__'),
    ],
  },
  {
    id: 'snapchat-ads',
    name: 'Snapchat Ads',
    category: 'social',
    params: [
      p('sub1', '{{site_source_name}}'),
      p('sub2', '{{campaign.id}}'),
      p('sub3', '{{adSet.id}}'),
      p('sub4', '{{ad.id}}'),
    ],
  },
  {
    id: 'x-ads',
    name: 'X (Twitter) Ads',
    category: 'social',
    params: [
      p('sub1', 'x'),
      p('sub2', '{{campaign.id}}'),
      p('sub3', '{{line_item.id}}'),
      p('sub4', '{{tweet.id}}'),
    ],
  },
  {
    id: 'pinterest-ads',
    name: 'Pinterest Ads',
    category: 'social',
    params: [
      p('sub1', 'pinterest'),
      p('sub2', '{campaignid}'),
      p('sub3', '{adgroupid}'),
      p('sub4', '{adid}'),
    ],
  },
  {
    id: 'linkedin-ads',
    name: 'LinkedIn Ads',
    category: 'social',
    params: [
      p('sub1', 'linkedin'),
      p('sub2', '{{CAMPAIGN_ID}}'),
      p('sub3', '{{CREATIVE_ID}}'),
    ],
  },
  {
    id: 'microsoft-ads',
    name: 'Microsoft Ads',
    category: 'search',
    params: [
      p('sub1', '{Network}'),
      p('sub2', '{CampaignId}'),
      p('sub3', '{AdGroupId}'),
      p('sub4', '{Keyword}'),
      p('gclid', '{msclkid}'),
    ],
  },
  {
    id: 'taboola',
    name: 'Taboola',
    category: 'native',
    params: [
      p('sub1', '{site}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{campaign_item_id}'),
      p('sub4', '{platform}'),
    ],
  },
  {
    id: 'outbrain',
    name: 'Outbrain',
    category: 'native',
    params: [
      p('sub1', '{{publisher_name}}'),
      p('sub2', '{{campaign_id}}'),
      p('sub3', '{{section_name}}'),
      p('sub4', '{{ad_id}}'),
    ],
  },
  {
    id: 'mgid',
    name: 'MGID',
    category: 'native',
    params: [
      p('sub1', '{widget_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{teaser_id}'),
      p('sub4', '{category_id}'),
    ],
  },
  {
    id: 'revcontent',
    name: 'Revcontent',
    category: 'native',
    params: [
      p('sub1', '{widget_id}'),
      p('sub2', '{adv_id}'),
      p('sub3', '{content_id}'),
    ],
  },
  {
    id: 'propellerads',
    name: 'PropellerAds',
    category: 'push',
    params: [
      p('sub1', '{zoneid}'),
      p('sub2', '{campaignid}'),
      p('sub3', '{bannerid}'),
      p('sub4', '{os}'),
    ],
  },
  {
    id: 'push-house',
    name: 'Push.house',
    category: 'push',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{creative_id}'),
    ],
  },
  {
    id: 'richads',
    name: 'RichAds',
    category: 'push',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{banner_id}'),
    ],
  },
  {
    id: 'adsterra',
    name: 'Adsterra',
    category: 'push',
    params: [
      p('sub1', '##PLACEMENT_ID##'),
      p('sub2', '##CAMPAIGN_ID##'),
      p('sub3', '##BANNER_ID##'),
    ],
  },
  {
    id: 'exoclick',
    name: 'ExoClick',
    category: 'adult',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{variation_id}'),
      p('sub4', '{zone_id}'),
    ],
  },
  {
    id: 'trafficjunky',
    name: 'TrafficJunky',
    category: 'adult',
    params: [
      p('sub1', '{siteid}'),
      p('sub2', '{campaignid}'),
      p('sub3', '{bannerid}'),
    ],
  },
  {
    id: 'juicyads',
    name: 'JuicyAds',
    category: 'adult',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{spot_id}'),
    ],
  },
  {
    id: 'trafficstars',
    name: 'TrafficStars',
    category: 'adult',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{creative_id}'),
    ],
  },
  {
    id: 'hilltopads',
    name: 'HilltopAds',
    category: 'push',
    params: [
      p('sub1', '{zoneid}'),
      p('sub2', '{campaignid}'),
      p('sub3', '{bannerid}'),
    ],
  },
  {
    id: 'zeropark',
    name: 'Zeropark',
    category: 'dsp',
    params: [
      p('sub1', '{target}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{visit_id}'),
    ],
  },
  {
    id: 'rollerads',
    name: 'RollerAds',
    category: 'push',
    params: [
      p('sub1', '{zone_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{banner_id}'),
    ],
  },
  {
    id: 'bidvertiser',
    name: 'BidVertiser',
    category: 'native',
    params: [
      p('sub1', '[SUBID]'),
      p('sub2', '[CAMPAIGNID]'),
    ],
  },
  {
    id: 'popcash',
    name: 'PopCash',
    category: 'push',
    params: [
      p('sub1', '[siteid]'),
      p('sub2', '[campaignid]'),
      p('sub3', '[category]'),
    ],
  },
  {
    id: 'popads',
    name: 'PopAds',
    category: 'push',
    params: [
      p('sub1', '[SITEID]'),
      p('sub2', '[CAMPAIGNID]'),
      p('sub3', '[FORMFACTOR]'),
    ],
  },
  {
    id: 'clickadu',
    name: 'Clickadu',
    category: 'push',
    params: [
      p('sub1', '{zoneid}'),
      p('sub2', '{campaignid}'),
      p('sub3', '{bannerid}'),
    ],
  },
  {
    id: 'evadav',
    name: 'Evadav',
    category: 'push',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', '{banner_id}'),
    ],
  },
  {
    id: 'richads-pop',
    name: 'RichAds Popunder',
    category: 'push',
    params: [
      p('sub1', '{site_id}'),
      p('sub2', '{campaign_id}'),
      p('sub3', 'pop'),
    ],
  },
  {
    id: 'direct-custom',
    name: 'Direct / Custom',
    category: 'direct',
    notes: 'Manual sub_ids — fill values or leave empty.',
    params: [],
  },
];

/**
 * Lookup template by id.
 */
export function trafficSourceById(id: string): TrafficSourceTemplate | null {
  for (let i = 0; i < TRAFFIC_SOURCE_TEMPLATES.length; i += 1) {
    if (TRAFFIC_SOURCE_TEMPLATES[i].id === id) return TRAFFIC_SOURCE_TEMPLATES[i];
  }
  return null;
}
