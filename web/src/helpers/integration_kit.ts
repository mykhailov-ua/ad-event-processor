export function normalizeTrackingHost(domain: string): string {
  return String(domain || '')
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
    .trim();
}

export function buildTrackPostbackURL(trackingDomain: string): string {
  const host = normalizeTrackingHost(trackingDomain) || 'track.example';
  return `https://${host}/track`;
}

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

export function buildInboundS2SCurl(trackURL: string, campaignId: string): string {
  const body = JSON.stringify({
    campaign_id: campaignId,
    type: 'conversion',
    click_id: 'REPLACE_CLICK_ID',
    user_id: 'REPLACE_USER_ID',
  });
  return `curl -sS -X POST '${trackURL}' -H 'Content-Type: application/json' -H 'Content-Length: ${body.length}' -d '${body}'`;
}

export function trafficGuideSummary(): string {
  return [
    'Click URL sends visitors through ad-event-processor (GET /click → 302 lander).',
    'When a sale happens, the affiliate or your CRM POSTs JSON to the postback URL (POST /track).',
    'Optional: lander JS fires the same /track without a redirect (zero-redirect).',
    'CAPI forwards settled conversions to Meta / Google / TikTok from the CAPI & Postbacks tab.',
  ].join(' ');
}
