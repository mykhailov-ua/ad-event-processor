/**
 * POST /track from a landing page (browser pixel). Requires TRACK_CORS_ORIGINS on tracker.
 * @param {Record<string, unknown>} opts
 * @returns {Promise<unknown>}
 */
export function trackEvent(opts) {
  const body = {
    campaign_id: opts.campaignId,
    type: opts.type,
  };
  if (opts.clickId) body.click_id = opts.clickId;
  if (opts.userId) body.user_id = opts.userId;
  const subs = opts.subs || {};
  for (let i = 1; i <= 30; i += 1) {
    const key = `sub${i}`;
    if (subs[key]) body[key] = subs[key];
  }
  const params = new URLSearchParams(window.location.search);
  for (const key of ['fbclid', 'gclid', 'ttclid']) {
    const val = params.get(key);
    if (val) body[key] = val;
  }
  return fetch(opts.endpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
    keepalive: true,
    credentials: 'omit',
  });
}

/**
 * HTML module snippet for zero-redirect conversion on a landing page.
 * @param {string} trackURL
 * @param {string} campaignId
 * @returns {string}
 */
export function buildDirectTrackSnippet(trackURL, campaignId) {
  return [
    '<script type="module">',
    `import { trackEvent } from '/src/static/track.js';`,
    'const params = new URLSearchParams(window.location.search);',
    'trackEvent({',
    `  endpoint: '${trackURL}',`,
    `  campaignId: '${campaignId}',`,
    "  type: 'conversion',",
    "  clickId: params.get('click_id') || '',",
    "  userId: params.get('user_id') || '',",
    '  subs: {',
    "    sub1: params.get('sub1') || '',",
    "    sub2: params.get('sub2') || '',",
    '  },',
    '});',
    '</script>',
  ].join('\n');
}
