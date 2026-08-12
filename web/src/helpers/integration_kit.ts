/**
 * Builders for campaign Integration kit (click / inbound S2S / conversion JSON).
 */

/**
 * Normalize tracking host (strip scheme / trailing slash).
 */
export function normalizeTrackingHost(domain: string): string {
  return String(domain || '')
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
    .trim();
}

/**
 * Absolute POST /track URL for conversions and affiliate inbound S2S.
 */
export function buildTrackPostbackURL(trackingDomain: string): string {
  const host = normalizeTrackingHost(trackingDomain) || 'track.example';
  return `https://${host}/track`;
}

/**
 * JSON body template affiliates / offer partners POST to BidShard on conversion.
 * Placeholders use common network token names; operators map them in the affiliate panel.
 */
export function buildInboundS2SBodyTemplate(campaignId: string): string {
  return [
    '{',
    `  "campaign_id": "${campaignId}",`,
    '  "type": "conversion",',
    '  "click_id": "{click_id}",',
    '  "user_id": "{user_id}",',
    '  "payout_micro": "{payout_micro}",',
    '  "sub1": "{sub1}",',
    '  "fbclid": "{fbclid}",',
    '  "gclid": "{gclid}",',
    '  "ttclid": "{ttclid}"',
    '}',
  ].join('\n');
}

/**
 * curl one-liner for testing inbound affiliate postback (replace tokens).
 */
export function buildInboundS2SCurl(trackURL: string, campaignId: string): string {
  const body = JSON.stringify({
    campaign_id: campaignId,
    type: 'conversion',
    click_id: 'REPLACE_CLICK_ID',
    user_id: 'REPLACE_USER_ID',
  });
  return `curl -sS -X POST '${trackURL}' -H 'Content-Type: application/json' -H 'Content-Length: ${body.length}' -d '${body}'`;
}

/**
 * Buyer-facing guide blurb (no eng-doc path dump).
 */
export function trafficGuideSummary(): string {
  return [
    'Click URL sends visitors through BidShard (GET /click → 302 lander).',
    'When a sale happens, the affiliate or your CRM POSTs JSON to the postback URL (POST /track).',
    'Optional: lander JS fires the same /track without a redirect (zero-redirect).',
    'CAPI forwards settled conversions to Meta / Google / TikTok from the CAPI & Postbacks tab.',
  ].join(' ');
}
