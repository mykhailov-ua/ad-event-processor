import type { ViewHandle } from '../lib/router_types.js';
import type { OpsDoctorSummary } from '../types/api/ops.js';
import { el, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { defaultClickTemplate } from '../helpers/tracking_link.js';
import {
  buildInboundS2SBodyTemplate,
  buildInboundS2SCurl,
  buildTrackPostbackURL,
  trafficGuideSummary,
} from '../helpers/integration_kit.js';
import { buildDirectTrackSnippet } from '../static/bidshard-track.js';
import { buildOpenRTBBidURL } from '../helpers/openrtb_endpoint.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { renderSectionCard } from '../ui/section_card.js';
import { renderButton } from '../ui/button.js';
import {
  TRAFFIC_SOURCE_TEMPLATES,
  templateParamMap,
  trafficSourceById,
  type TrafficSourceTemplate,
} from '../models/traffic_source_templates.js';
import { buildTemplatedClickURL } from '../helpers/traffic_source_url.js';

export type CampaignTrackingPanelOpts = {
  campaignId: string;
  navigate?: (path: string) => void;
};

type PlatformState = {
  clickTemplate: string;
  trackingDomain: string;
  edgeClick: boolean;
  edgeOpenRTB: boolean;
  rtbMode: string;
  rtbEnabled: boolean;
};

/**
 * Copy text to the clipboard and show a toast.
 */
function copyText(label: string, text: string): void {
  navigator.clipboard?.writeText(text).then(() => {
    pushToastMessage({ title: 'Copied', message: `${label} copied to clipboard` });
  }).catch(() => {
    pushToastMessage({ title: 'Copy failed', message: text });
  });
}

/**
 * Render a copyable code block with label.
 */
function copyRow(label: string, value: string, testId?: string): HTMLElement {
  return el('div', {
    className: 'integration-copy-row',
    ...(testId ? { 'data-testid': testId } : {}),
  },
    el('div', { className: 'integration-copy-row__head' },
      el('span', { className: 'form-label' }, label),
      renderButton({
        label: 'Copy',
        variant: 'secondary',
        size: 'sm',
        testId: testId ? `${testId}-copy` : undefined,
        onClick: () => copyText(label, value),
      }),
    ),
    el('code', { className: 'code-block' }, value),
  );
}

/**
 * Mount per-campaign traffic integration kit (click URL, templates, postback, macros).
 */
export function mountCampaignTrackingPanel(
  container: HTMLElement,
  opts: CampaignTrackingPanelOpts,
): ViewHandle {
  let destroyed = false;
  const params: Record<string, string> = {};
  let selectedTemplateId = 'direct-custom';
  let platform: PlatformState = {
    clickTemplate: defaultClickTemplate(''),
    trackingDomain: '',
    edgeClick: false,
    edgeOpenRTB: false,
    rtbMode: 'off',
    rtbEnabled: false,
  };

  function applyTemplate(tpl: TrafficSourceTemplate): void {
    const map = templateParamMap(tpl);
    const nextKeys = new Set(Object.keys(map));
    for (const key of Object.keys(params)) {
      if (!nextKeys.has(key)) params[key] = '';
    }
    for (const [k, v] of Object.entries(map)) {
      params[k] = v;
    }
  }

  async function loadTemplate(): Promise<void> {
    const [docRes, platRes] = await Promise.all([
      to(api<OpsDoctorSummary>('/api/v1/ops/doctor')),
      to(api<{
        config?: {
          tracking_domain?: string;
          edge_expose_click?: boolean;
          edge_expose_openrtb?: boolean;
        };
        click_url_template?: string;
      }>('/api/v1/settings/platform')),
    ]);
    if (destroyed) return;
    const doc = docRes[0]?.data;
    const plat = platRes[0]?.data;
    platform = {
      trackingDomain: plat?.config?.tracking_domain ?? doc?.tracking_domain ?? '',
      edgeClick: plat?.config?.edge_expose_click ?? false,
      edgeOpenRTB: plat?.config?.edge_expose_openrtb ?? false,
      rtbMode: String(doc?.rtb_mode || 'off').toLowerCase(),
      rtbEnabled: doc?.rtb_enabled === true,
      clickTemplate: doc?.click_url_template
        || plat?.click_url_template
        || defaultClickTemplate(plat?.config?.tracking_domain ?? doc?.tracking_domain ?? ''),
    };
    render();
  }

  function goPostbacks(e: Event): void {
    e.preventDefault();
    const path = `/campaigns/${opts.campaignId}?tab=postbacks`;
    if (opts.navigate) {
      opts.navigate(path);
      return;
    }
    const url = new URL(window.location.href);
    url.searchParams.set('tab', 'postbacks');
    window.history.pushState({}, '', url);
    window.dispatchEvent(new PopStateEvent('popstate'));
  }

  function render(): void {
    const link = buildTemplatedClickURL(
      platform.clickTemplate || platform.trackingDomain,
      opts.campaignId,
      params,
    );
    const trackURL = buildTrackPostbackURL(platform.trackingDomain);
    const openrtbURL = buildOpenRTBBidURL(platform.trackingDomain || 'track.example');
    const directSnippet = buildDirectTrackSnippet(trackURL, opts.campaignId);
    const inboundBody = buildInboundS2SBodyTemplate(opts.campaignId);
    const inboundCurl = buildInboundS2SCurl(trackURL, opts.campaignId);
    const impressionJSON = [
      '{',
      `  "campaign_id": "${opts.campaignId}",`,
      '  "type": "impression",',
      '  "user_id": "{user_id}"',
      '}',
    ].join('\n');
    const selected = trafficSourceById(selectedTemplateId);

    const editableKeys = [
      'sub1', 'sub2', 'sub3', 'sub4', 'sub5', 'sub6', 'sub7', 'sub8', 'sub9', 'sub10',
      'ad_campaign_id', 'fbclid', 'gclid', 'ttclid',
    ];
    const extendedSubKeys = Array.from({ length: 20 }, (_, i) => `sub${i + 11}`);

    container.replaceChildren(
      el('div', { className: 'stack integration-kit', 'data-testid': 'campaign-integration-kit' },
        el('p', { className: 'text-muted text-sm', 'data-testid': 'integration-intro' },
          trafficGuideSummary(),
          ' ',
          el('a', {
            href: `/campaigns/${opts.campaignId}?tab=postbacks`,
            className: 'text-sm',
            onClick: goPostbacks,
          }, 'CAPI & Postbacks →'),
        ),
        el('details', { className: 'integration-guide', 'data-testid': 'traffic-guide' },
          el('summary', null, 'Traffic integration guide (buyer)'),
          el('div', { className: 'stack text-sm' },
            el('p', null,
              'Full wire contracts ship with the appliance as ',
              el('code', { className: 'code-inline' }, 'docs/TRAFFIC_INTEGRATION.md'),
              ' (also in the operator bundle). Summary:',
            ),
            el('ul', { className: 'list-plain' },
              el('li', null, 'Campaign traffic → Click URL (GET /click).'),
              el('li', null, 'Affiliate / CRM conversion → inbound S2S POST /track (JSON, Content-Length required).'),
              el('li', null, 'Lander pixel → zero-redirect fetch() to the same /track URL.'),
              el('li', null, 'Ad platforms (Meta/Google/TikTok) → configure on CAPI & Postbacks; BidShard forwards after settlement.'),
            ),
            el('p', { className: 'text-muted' },
              'Edge tip: enable “Expose click URL on edge” in Platform settings so buyers hit :443 instead of tracker ports.',
            ),
          ),
        ),
        renderSectionCard({
          icon: 'shield',
          title: 'Safe page vs money URL',
          desc: 'Lightweight cloak companion: suspicious traffic can 302 to a safe URL while clean clicks reach brand landings.',
          children: [
            el('p', { className: 'text-muted text-sm', 'data-testid': 'integration-safe-page-hint' },
              'Money URL: weighted brand creatives (Creative tab) or campaign target URL. ',
              'Safe URL: white-page for IVT / placement blacklist hits when enabled under Configuration.',
            ),
            el('p', null,
              el('a', {
                href: `/campaigns/${opts.campaignId}?tab=config`,
                className: 'text-sm',
                onClick: (e: Event) => {
                  if (!opts.navigate) return;
                  e.preventDefault();
                  opts.navigate(`/campaigns/${opts.campaignId}?tab=config`);
                },
              }, 'Configure safe page →'),
            ),
          ],
        }),
        renderSectionCard({
          icon: 'globe',
          title: 'Click URL (campaign traffic)',
          desc: platform.edgeClick
            ? 'Served on the edge (:8180/:443) with shard routing. Pick a traffic-source template to pre-fill network macros.'
            : 'Enable “Expose click URL on edge” in Platform settings, or point traffic at tracker ports :8181–8184.',
          children: [
            el('label', {
              className: 'form-field',
              htmlFor: 'traffic-source-template',
              'data-testid': 'traffic-source-template-field',
            },
              'Traffic source template',
              el('select', {
                id: 'traffic-source-template',
                className: 'form-input form-input--sm',
                'data-testid': 'traffic-source-template',
                value: selectedTemplateId,
                onChange: (e: Event) => {
                  selectedTemplateId = eventTargetValue(e);
                  const tpl = trafficSourceById(selectedTemplateId);
                  if (tpl) applyTemplate(tpl);
                  render();
                },
              },
                ...TRAFFIC_SOURCE_TEMPLATES.map((tpl) => el('option', {
                  value: tpl.id,
                  selected: tpl.id === selectedTemplateId,
                }, tpl.name)),
              ),
            ),
            selected?.notes
              ? el('p', { className: 'text-muted text-sm', 'data-testid': 'traffic-source-notes' }, selected.notes)
              : null,
            selected?.cost_sync
              ? el('p', { className: 'text-muted text-sm' },
                `Cost Sync network: ${selected.cost_sync}. Keep ad_campaign_id / sub2 as the external campaign id.`,
              )
              : null,
            ...editableKeys.map((key) => el('label', { className: 'form-field', htmlFor: `track-${key}` },
              key,
              el('input', {
                id: `track-${key}`,
                className: 'form-input form-input--sm',
                placeholder: key.startsWith('sub') ? `Value or macro for ${key}` : `Optional ${key}`,
                value: params[key] ?? '',
                onInput: (e: Event) => {
                  params[key] = eventTargetValue(e);
                  render();
                },
              }),
            )),
            el('details', { className: 'integration-sub-extend', 'data-testid': 'integration-sub11-30' },
              el('summary', null, 'Sub 11–30'),
              el('div', { className: 'stack mt-2' },
                ...extendedSubKeys.map((key) => el('label', { className: 'form-field', htmlFor: `track-${key}` },
                  key,
                  el('input', {
                    id: `track-${key}`,
                    className: 'form-input form-input--sm',
                    placeholder: `{${key}}`,
                    value: params[key] ?? '',
                    onInput: (e: Event) => {
                      params[key] = eventTargetValue(e);
                      render();
                    },
                  }),
                )),
              ),
            ),
            copyRow('Click URL', link, 'integration-click-url'),
          ],
        }),
        renderSectionCard({
          icon: 'download',
          title: 'Affiliate inbound S2S postback',
          desc: 'Give this URL to the affiliate network or offer partner. They POST JSON when a conversion settles — distinct from CAPI outbound to Meta/Google/TikTok.',
          children: [
            copyRow('Postback URL', trackURL, 'integration-inbound-url'),
            copyRow('JSON body template', inboundBody, 'integration-inbound-body'),
            copyRow('Test with curl', inboundCurl, 'integration-inbound-curl'),
            el('p', { className: 'text-muted text-sm' },
              'Map network tokens to BidShard fields: click id → click_id, payout → payout_micro (micro-units) or omit and settle later. ',
              'Requires Content-Length; chunked encoding is rejected on /track.',
            ),
          ],
        }),
        renderSectionCard({
          icon: 'code',
          title: 'Zero-redirect (browser pixel)',
          desc: 'Fire /track from the landing page via fetch(). Set TRACK_CORS_ORIGINS on the tracker to include your LP origin.',
          children: [
            copyRow('Landing page snippet', directSnippet, 'integration-direct-snippet'),
            el('p', { className: 'text-muted text-sm' },
              'Module loads bidshard-track.js; auto-picks fbclid/gclid/ttclid from the page query string.',
            ),
          ],
        }),
        renderSectionCard({
          icon: 'eye',
          title: 'Impression / event examples',
          desc: 'Same POST /track URL as inbound S2S. Native JSON or protobuf (Content-Type application/x-protobuf).',
          children: [
            copyRow('Impression JSON example', impressionJSON, 'integration-impression-json'),
            el('p', { className: 'text-muted text-sm' },
              'Conversion body is in the Affiliate inbound S2S section above.',
            ),
          ],
        }),
        renderRtbTrackSection(platform, trackURL),
        platform.edgeOpenRTB
          ? renderSectionCard({
            icon: 'activity',
            title: 'OpenRTB exchange (SSP partners)',
            desc: 'Separate from /track RTB_MODE. SSPs POST OpenRTB 2.6 bid requests here — not the SDK /track path.',
            children: [
              copyRow('Bid endpoint', openrtbURL, 'integration-openrtb-url'),
              el('p', null,
                el('a', { href: '/rtb/integration', className: 'text-sm' }, 'RTB integration profile & validate-bid →'),
              ),
            ],
          })
          : el('p', { className: 'text-muted text-sm' },
            'OpenRTB exchange: enable “Expose OpenRTB bid endpoint on edge” in Platform settings, or point partners at tracker :8181–8184/openrtb/bid. ',
            el('a', { href: '/rtb/integration' }, 'Integration onboarding'),
          ),
        renderSectionCard({
          icon: 'file-spreadsheet',
          title: 'Macro reference',
          desc: 'Substitute in landing URLs and partner postbacks.',
          children: [
            el('table', { className: 'data-table data-table--compact', 'data-testid': 'integration-macro-table' },
              el('thead', null,
                el('tr', null,
                  el('th', null, 'Macro'),
                  el('th', null, 'Meaning'),
                ),
              ),
              el('tbody', null,
                ...([
                  ['{campaign_id}', 'Campaign UUID'],
                  ['{click_id}', 'Unique click id (generated on redirect)'],
                  ['{user_id}', 'Publisher user / visitor id'],
                  ['{sub1}…{sub30}', 'Arbitrary sub-ids for source tracking'],
                  ['{subid1}…{subid30}', 'Partner postback macro aliases'],
                  ['gclid, ttclid, fbclid', 'Attribution IDs — stored on event + forwarded to lander'],
                  ['ad_campaign_id', 'Network campaign id for Cost Sync join (often mirrors sub2)'],
                ] as const).map(([macro, meaning]) => el('tr', null,
                  el('td', null, el('code', { className: 'code-inline' }, macro)),
                  el('td', null, meaning),
                )),
              ),
            ),
          ],
        }),
      ),
    );
  }

  render();
  loadTemplate();

  return {
    destroy() {
      destroyed = true;
    },
  };
}

/**
 * Render RTB-on-/track explainer (RTB_MODE), distinct from OpenRTB exchange endpoint.
 */
function renderRtbTrackSection(platform: PlatformState, trackURL: string): HTMLElement {
  const mode = platform.rtbMode || 'off';
  const modeDesc: Record<string, string> = {
    off: 'In-process auction on /track is disabled. campaign_id must be present in the postback body.',
    shadow: 'Auction runs before filters for metrics only — campaign_id in the body is unchanged. Use before promoting to live.',
    live: 'Auction may rewrite campaign_id to the winning line item before FilterEngine. No-bid rejects skip filters.',
  };
  const desc = modeDesc[mode] || modeDesc.off;
  const children: Array<HTMLElement | null> = [
    el('p', { className: 'text-muted text-sm' },
      'SDK / single-endpoint flows only. Exchange partners use POST /openrtb/bid — see section below. ',
      'Wire comparison is in the appliance guide docs/TRAFFIC_INTEGRATION.md §2.1.',
    ),
    el('dl', { className: 'definition-list' },
      el('dt', null, 'RTB_MODE (tracker)'),
      el('dd', { className: 'font-mono' }, platform.rtbEnabled ? mode : 'off (RTB disabled)'),
      el('dt', null, 'Auction URL'),
      el('dd', { className: 'font-mono' }, trackURL),
    ),
    el('p', { className: 'text-muted text-sm' }, desc),
  ];
  if (!platform.rtbEnabled) {
    children.push(el('p', { className: 'text-muted text-sm' },
      'RTB is not enabled on this deployment (license or RTB_MODE=off).',
    ));
  }
  return renderSectionCard({
    icon: 'zap',
    title: 'In-app auction on /track',
    desc: 'Optional RTB_MODE=shadow|live — not a replacement for the OpenRTB exchange endpoint.',
    children,
  });
}
