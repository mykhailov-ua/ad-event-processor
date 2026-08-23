/**
 * POST a conversion or event to the tracker /track endpoint (zero-redirect).
 */
export function adEventProcessorTrack(opts) {
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
 * Build an inline module script that calls adEventProcessorTrack on the current page.
 */
export function buildDirectTrackSnippet(trackURL, campaignId) {
  return [
    '<script type="module">',
    `import { adEventProcessorTrack } from '/src/static/ad-event-processor-track.js';`,
    'const params = new URLSearchParams(window.location.search);',
    'adEventProcessorTrack({',
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
