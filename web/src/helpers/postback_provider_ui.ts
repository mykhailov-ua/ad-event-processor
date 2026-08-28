export type PostbackProvider =
  | 'webhook'
  | 'facebook'
  | 'google'
  | 'tiktok'
  | 'taboola'
  | 'outbrain'
  | 'microsoft_ads';

export type PostbackProviderUi = {
  label: string;
  blurb: string;
  primaryLabel: string;
  primaryPlaceholder: string;
  primaryHelp: string;
  tokenLabel: string;
  tokenPlaceholder: string;
  tokenHelp: string;
  requiresToken: boolean;
  showMacros: boolean;
  supportsTestEventCode: boolean;
  eventMappingHint: string;
};

export const POSTBACK_PROVIDER_UI: Record<PostbackProvider, PostbackProviderUi> = {
  webhook: {
    label: 'Webhook (affiliate S2S)',
    blurb:
      'GET request with macro-expanded query string. Use for affiliate networks and custom partners.',
    primaryLabel: 'URL template',
    primaryPlaceholder: 'https://partner.example/postback?cid={click_id}&payout={payout}',
    primaryHelp:
      'Macros: {click_id}, {payout}, {tx_id}, {event_type}, {sub1}...{sub30}, {subid1}...{subid30}',
    tokenLabel: 'Bearer token (optional)',
    tokenPlaceholder: 'Leave blank to keep saved token',
    tokenHelp: 'Sent as Authorization: Bearer ... when set.',
    requiresToken: false,
    showMacros: true,
    supportsTestEventCode: false,
    eventMappingHint:
      'Outbound webhook keeps ad-event-processor event_type; map names in the partner template if needed.',
  },
  facebook: {
    label: 'Meta Conversions API',
    blurb:
      'Server-side events to Graph API. Requires fbclid on the conversion (from click URL or /track payload).',
    primaryLabel: 'Pixel ID',
    primaryPlaceholder: '123456789012345',
    primaryHelp:
      'Numeric Meta pixel ID. Stored as the postback template; Graph URL is built automatically.',
    tokenLabel: 'CAPI access token',
    tokenPlaceholder: 'EAA...',
    tokenHelp:
      'System user or long-lived token with ads_management. Encrypted at rest; never shown after save.',
    requiresToken: true,
    showMacros: false,
    supportsTestEventCode: true,
    eventMappingHint:
      'ad-event-processor -> Meta: conversion/purchase -> Purchase; lead -> Lead; install -> CompleteRegistration; click -> ViewContent.',
  },
  google: {
    label: 'Google Ads offline conversions',
    blurb: 'Uploads gclid-based conversions. Requires gclid on the conversion event payload.',
    primaryLabel: 'Conversion action resource',
    primaryPlaceholder: 'customers/1234567890/conversionActions/987654321',
    primaryHelp: 'Full Google Ads resource name for the conversion action (not the display name).',
    tokenLabel: 'OAuth access token',
    tokenPlaceholder: 'ya29...',
    tokenHelp:
      'Short-lived OAuth 2.0 access token with https://www.googleapis.com/auth/adwords scope.',
    requiresToken: true,
    showMacros: false,
    supportsTestEventCode: false,
    eventMappingHint:
      'Google uses the conversion action resource above; ad-event-processor event_type selects when to fire, not the Ads action name.',
  },
  tiktok: {
    label: 'TikTok Events API',
    blurb: 'Server-side events to TikTok Business API. Requires ttclid on the conversion payload.',
    primaryLabel: 'Pixel code',
    primaryPlaceholder: 'CXXXXXXXXXXXXXXX',
    primaryHelp: 'Events API pixel code from TikTok Ads Manager.',
    tokenLabel: 'Access token',
    tokenPlaceholder: 'Leave blank to keep saved token',
    tokenHelp: 'Sent as Access-Token header. Generate under Tools -> Events API.',
    requiresToken: true,
    showMacros: false,
    supportsTestEventCode: true,
    eventMappingHint:
      'ad-event-processor -> TikTok: conversion/purchase -> CompletePayment; lead -> Contact; install -> Download; click -> ClickButton.',
  },
  taboola: {
    label: 'Taboola S2S',
    blurb:
      'Server-side conversion GET to Taboola trc endpoint. Requires tblci (or click_id) on the conversion payload.',
    primaryLabel: 'Taboola event name',
    primaryPlaceholder: 'registration',
    primaryHelp:
      'Exact Event Name from Taboola Realize conversion settings (not the conversion display label).',
    tokenLabel: 'Bearer token (optional)',
    tokenPlaceholder: 'Leave blank unless Taboola issued an API token',
    tokenHelp: 'Most S2S setups use unsigned GET postbacks; token is optional.',
    requiresToken: false,
    showMacros: false,
    supportsTestEventCode: false,
    eventMappingHint:
      'Taboola expects click-id query param from tblci on the landing URL. Map payout to revenue when set.',
  },
  outbrain: {
    label: 'Outbrain S2S',
    blurb:
      'Server-side conversion GET to tr.outbrain.com/unifiedPixel. Requires ob_click_id on the conversion payload.',
    primaryLabel: 'Outbrain conversion name',
    primaryPlaceholder: 'Purchase',
    primaryHelp:
      'Exact event-based conversion name from Amplify (must match postback name= parameter).',
    tokenLabel: 'Bearer token (optional)',
    tokenPlaceholder: 'Leave blank for default S2S pixel',
    tokenHelp: 'Standard Outbrain S2S uses unsigned GET requests.',
    requiresToken: false,
    showMacros: false,
    supportsTestEventCode: false,
    eventMappingHint:
      'Capture ob_click_id (or obclid) from the click URL. Optional orderValue from payout_micro.',
  },
  microsoft_ads: {
    label: 'Microsoft Ads offline conversions',
    blurb: 'ApplyOfflineConversions REST upload. Requires msclkid on the conversion payload.',
    primaryLabel: 'Account | customer | conversion goal',
    primaryPlaceholder: '123456789|987654321|My Offline Goal',
    primaryHelp:
      'Pipe-separated advertising account id, customer id, and offline conversion goal name.',
    tokenLabel: 'OAuth access token',
    tokenPlaceholder: 'ya29... or JSON with access_token',
    tokenHelp: 'OAuth 2.0 access token with https://ads.microsoft.com/msads.manage scope.',
    requiresToken: true,
    showMacros: false,
    supportsTestEventCode: true,
    eventMappingHint:
      'Store Microsoft developer token in Test event code field. Conversion goal name must match OfflineConversionGoal in Ads UI.',
  },
};

export function postbackProviderIds(): PostbackProvider[] {
  return Object.keys(POSTBACK_PROVIDER_UI) as PostbackProvider[];
}

export function normalizePostbackProvider(provider: string): PostbackProvider {
  const id = (provider || 'webhook').toLowerCase();
  return id in POSTBACK_PROVIDER_UI ? (id as PostbackProvider) : 'webhook';
}
