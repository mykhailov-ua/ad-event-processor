export function buildOpenRTBBidURL(host: string): string {
  const h = (host || '')
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '')
    .trim();
  if (!h) {
    return 'https://track.example/openrtb/bid';
  }
  return `https://${h}/openrtb/bid`;
}

export function openRTBRoutingHint(opts: {
  edgeExposeOpenRTB?: boolean;
  edgePortHint?: string;
  trackerPortHint?: string;
}): string {
  if (opts.edgeExposeOpenRTB) {
    return `Edge nginx${opts.edgePortHint ?? ':8180'} (platform setting enabled)`;
  }
  return `Tracker ports${opts.trackerPortHint ?? ':8181–8184'} (enable edge OpenRTB in Platform settings for :8180)`;
}

export const VALIDATE_BID_FIXTURE = {
  id: 'req-smoke-001',
  tmax: 250,
  cur: ['USD'],
  imp: [
    {
      id: 'imp-1',
      bidfloor: 1.25,
      banner: { w: 300, h: 250 },
    },
  ],
  site: { domain: 'example.com', page: 'https://example.com/' },
  device: {
    ip: '203.0.113.1',
    ua: 'Mozilla/5.0',
    devicetype: 2,
    geo: { country: 'USA' },
  },
};
