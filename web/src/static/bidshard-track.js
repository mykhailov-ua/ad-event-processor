/**
 * Zero-redirect browser pixel: POST conversion events to /track from the landing page.
 *
 * @param {object} opts
 * @param {string} opts.endpoint - Full /track URL (https://track.example/track)
 * @param {string} opts.campaignId - Campaign UUID
 * @param {string} opts.type - Event type (e.g. conversion, impression)
 * @param {string} [opts.clickId] - Click id from query string
 * @param {string} [opts.userId] - Publisher user id
 * @param {Record<string, string>} [opts.subs] - sub1..sub30 values
 */
export function bidshardTrack(opts) {
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
 * Build a copy-paste snippet for the integration panel.
 *
 * @param {string} trackURL
 * @param {string} campaignId
 * @returns {string}
 */
export function buildDirectTrackSnippet(trackURL, campaignId) {
  return [
    '<script type="module">',
    `import { bidshardTrack } from '/src/static/bidshard-track.js';`,
    'const params = new URLSearchParams(window.location.search);',
    'bidshardTrack({',
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
