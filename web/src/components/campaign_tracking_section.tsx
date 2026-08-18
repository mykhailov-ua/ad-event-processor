import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import type { OpsDoctorSummary } from '../types/ops.js';
import { defaultClickTemplate } from '../helpers/tracking_link.js';
import {
  buildInboundS2SBodyTemplate,
  buildInboundS2SCurl,
  buildTrackPostbackURL,
  trafficGuideSummary,
} from '../helpers/integration_kit.js';
import { buildDirectTrackSnippet } from '../static/bidshard-track.js';
import { buildOpenRTBBidURL } from '../helpers/openrtb_endpoint.js';
import {
  TRAFFIC_SOURCE_TEMPLATES,
  templateParamMap,
  trafficSourceById,
} from '../models/traffic_source_templates.js';
import { buildTemplatedClickURL } from '../helpers/traffic_source_url.js';
import { SectionCard } from './section_card.js';
import { IntegrationCopyRow } from './integration_copy_row.js';
import { CampaignApplyTemplatesPanel } from './campaign_apply_templates_panel.js';

type PlatformState = {
  clickTemplate: string;
  trackingDomain: string;
  edgeClick: boolean;
  edgeOpenRTB: boolean;
  rtbMode: string;
  rtbEnabled: boolean;
};

const EDITABLE_KEYS = [
  'sub1',
  'sub2',
  'sub3',
  'sub4',
  'sub5',
  'sub6',
  'sub7',
  'sub8',
  'sub9',
  'sub10',
  'ad_campaign_id',
  'fbclid',
  'gclid',
  'ttclid',
];

const EXTENDED_SUB_KEYS = Array.from({ length: 20 }, (_, i) => `sub${i + 11}`);

export type CampaignTrackingSectionProps = {
  campaignId: string;
  canWrite?: boolean;
};

function RtbTrackSection({ platform, trackURL }: { platform: PlatformState; trackURL: string }) {
  const mode = platform.rtbMode || 'off';
  const modeDesc: Record<string, string> = {
    off: 'In-process auction on /track is disabled. campaign_id must be present in the postback body.',
    shadow:
      'Auction runs before filters for metrics only — campaign_id in the body is unchanged. Use before promoting to live.',
    live: 'Auction may rewrite campaign_id to the winning line item before FilterEngine. No-bid rejects skip filters.',
  };
  const desc = modeDesc[mode] || modeDesc.off;

  return (
    <SectionCard
      icon="zap"
      title="In-app auction on /track"
      desc="Optional RTB_MODE=shadow|live — not a replacement for the OpenRTB exchange endpoint."
    >
      <p className="text-muted text-sm">
        SDK / single-endpoint flows only. Exchange partners use POST /openrtb/bid — see section
        below. Wire comparison is in the appliance guide docs/TRAFFIC_INTEGRATION.md §2.1.
      </p>
      <dl className="definition-list">
        <dt>RTB_MODE (tracker)</dt>
        <dd className="font-mono">{platform.rtbEnabled ? mode : 'off (RTB disabled)'}</dd>
        <dt>Auction URL</dt>
        <dd className="font-mono">{trackURL}</dd>
      </dl>
      <p className="text-muted text-sm">{desc}</p>
      {!platform.rtbEnabled ? (
        <p className="text-muted text-sm">
          RTB is not enabled on this deployment (license or RTB_MODE=off).
        </p>
      ) : null}
    </SectionCard>
  );
}

export function CampaignTrackingSection({
  campaignId,
  canWrite = false,
}: CampaignTrackingSectionProps) {
  const [params, setParams] = useState<Record<string, string>>({});
  const [selectedTemplateId, setSelectedTemplateId] = useState('direct-custom');
  const [dmrEnabled, setDmrEnabled] = useState(false);
  const [utmEnabled, setUtmEnabled] = useState(false);
  const [utmSource, setUtmSource] = useState('');
  const [utmMedium, setUtmMedium] = useState('');
  const [utmCampaign, setUtmCampaign] = useState('');
  const [platform, setPlatform] = useState<PlatformState>({
    clickTemplate: defaultClickTemplate(''),
    trackingDomain: '',
    edgeClick: false,
    edgeOpenRTB: false,
    rtbMode: 'off',
    rtbEnabled: false,
  });

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const [docRes, platRes] = await Promise.all([
        to(api<OpsDoctorSummary>('/api/v1/ops/doctor')),
        to(
          api<{
            config?: {
              tracking_domain?: string;
              edge_expose_click?: boolean;
              edge_expose_openrtb?: boolean;
            };
            click_url_template?: string;
          }>('/api/v1/settings/platform')
        ),
      ]);
      if (cancelled) return;
      const doc = docRes[0]?.data;
      const plat = platRes[0]?.data;
      setPlatform({
        trackingDomain: plat?.config?.tracking_domain ?? doc?.tracking_domain ?? '',
        edgeClick: plat?.config?.edge_expose_click ?? false,
        edgeOpenRTB: plat?.config?.edge_expose_openrtb ?? false,
        rtbMode: String(doc?.rtb_mode || 'off').toLowerCase(),
        rtbEnabled: doc?.rtb_enabled === true,
        clickTemplate:
          doc?.click_url_template ||
          plat?.click_url_template ||
          defaultClickTemplate(plat?.config?.tracking_domain ?? doc?.tracking_domain ?? ''),
      });
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const selected = trafficSourceById(selectedTemplateId);

  const link = useMemo(
    () =>
      buildTemplatedClickURL(
        platform.clickTemplate || platform.trackingDomain,
        campaignId,
        params,
        {
          dmr: dmrEnabled,
          utm: utmEnabled
            ? {
                utm_source: utmSource || undefined,
                utm_medium: utmMedium || undefined,
                utm_campaign: utmCampaign || campaignId,
              }
            : undefined,
        }
      ),
    [
      platform.clickTemplate,
      platform.trackingDomain,
      campaignId,
      params,
      dmrEnabled,
      utmEnabled,
      utmSource,
      utmMedium,
      utmCampaign,
    ]
  );

  const trackURL = buildTrackPostbackURL(platform.trackingDomain);
  const openrtbURL = buildOpenRTBBidURL(platform.trackingDomain || 'track.example');
  const directSnippet = buildDirectTrackSnippet(trackURL, campaignId);
  const inboundBody = buildInboundS2SBodyTemplate(campaignId);
  const inboundCurl = buildInboundS2SCurl(trackURL, campaignId);
  const impressionJSON = [
    '{',
    `  "campaign_id": "${campaignId}",`,
    '  "type": "impression",',
    '  "user_id": "{user_id}"',
    '}',
  ].join('\n');

  const applyTemplate = (templateId: string) => {
    setSelectedTemplateId(templateId);
    const tpl = trafficSourceById(templateId);
    if (!tpl) return;
    const map = templateParamMap(tpl);
    setParams((prev) => {
      const next: Record<string, string> = {};
      for (const key of Object.keys(prev)) {
        if (!(key in map)) next[key] = '';
      }
      for (const [k, v] of Object.entries(map)) {
        next[k] = v;
      }
      return { ...prev, ...next };
    });
  };

  const setParam = (key: string, value: string) => {
    setParams((prev) => ({ ...prev, [key]: value }));
  };

  return (
    <div className="stack integration-kit" data-testid="campaign-integration-kit">
      <p className="text-muted text-sm" data-testid="integration-intro">
        {trafficGuideSummary()}{' '}
        <Link to={`/campaigns/${campaignId}?tab=postbacks`} className="text-sm">
          CAPI & Postbacks →
        </Link>
      </p>

      <details className="integration-guide" data-testid="traffic-guide">
        <summary>Traffic integration guide (buyer)</summary>
        <div className="stack text-sm">
          <p>
            Full wire contracts ship with the appliance as{' '}
            <code className="code-inline">docs/TRAFFIC_INTEGRATION.md</code> (also in the operator
            bundle). Summary:
          </p>
          <ul className="list-plain">
            <li>Campaign traffic → Click URL (GET /click).</li>
            <li>
              Affiliate / CRM conversion → inbound S2S POST /track (JSON, Content-Length required).
            </li>
            <li>Lander pixel → zero-redirect fetch() to the same /track URL.</li>
            <li>
              Ad platforms (Meta/Google/TikTok) → configure on CAPI & Postbacks; BidShard forwards
              after settlement.
            </li>
          </ul>
          <p className="text-muted">
            Edge tip: enable “Expose click URL on edge” in Platform settings so buyers hit :443
            instead of tracker ports.
          </p>
        </div>
      </details>

      <SectionCard
        icon="shield"
        title="Safe page vs money URL"
        desc="Lightweight cloak companion: suspicious traffic can 302 to a safe URL while clean clicks reach brand landings."
      >
        <p className="text-muted text-sm" data-testid="integration-safe-page-hint">
          Money URL: weighted brand creatives (Creative tab) or campaign target URL. Safe URL:
          white-page for IVT / placement blacklist hits when enabled under Configuration.
        </p>
        <p>
          <Link to={`/campaigns/${campaignId}?tab=config`} className="text-sm">
            Configure safe page →
          </Link>
        </p>
      </SectionCard>

      <SectionCard
        icon="globe"
        title="Click URL (campaign traffic)"
        desc={
          platform.edgeClick
            ? 'Served on the edge (:8180/:443) with shard routing. Pick a traffic-source template to pre-fill network macros.'
            : 'Enable “Expose click URL on edge” in Platform settings, or point traffic at tracker ports :8181–8184.'
        }
      >
        <label
          className="form-field"
          htmlFor="traffic-source-template"
          data-testid="traffic-source-template-field"
        >
          Traffic source template
          <select
            id="traffic-source-template"
            className="form-input form-input--sm"
            data-testid="traffic-source-template"
            value={selectedTemplateId}
            onChange={(e) => applyTemplate(e.target.value)}
          >
            {TRAFFIC_SOURCE_TEMPLATES.map((tpl) => (
              <option key={tpl.id} value={tpl.id}>
                {tpl.name}
              </option>
            ))}
          </select>
        </label>
        {selected?.notes ? (
          <p className="text-muted text-sm" data-testid="traffic-source-notes">
            {selected.notes}
          </p>
        ) : null}
        {selected?.cost_sync ? (
          <p className="text-muted text-sm">
            {`Cost Sync network: ${selected.cost_sync}. Keep ad_campaign_id / sub2 as the external campaign id.`}
          </p>
        ) : null}
        {EDITABLE_KEYS.map((key) => (
          <label key={key} className="form-field" htmlFor={`track-${key}`}>
            {key}
            <input
              id={`track-${key}`}
              className="form-input form-input--sm"
              placeholder={key.startsWith('sub') ? `Value or macro for ${key}` : `Optional ${key}`}
              value={params[key] ?? ''}
              onChange={(e) => setParam(key, e.target.value)}
            />
          </label>
        ))}
        <details className="integration-sub-extend" data-testid="integration-sub11-30">
          <summary>Sub 11–30</summary>
          <div className="stack mt-2">
            {EXTENDED_SUB_KEYS.map((key) => (
              <label key={key} className="form-field" htmlFor={`track-${key}`}>
                {key}
                <input
                  id={`track-${key}`}
                  className="form-input form-input--sm"
                  placeholder={`{${key}}`}
                  value={params[key] ?? ''}
                  onChange={(e) => setParam(key, e.target.value)}
                />
              </label>
            ))}
          </div>
        </details>
        <div className="toolbar-row" data-testid="integration-click-url-options">
          <label className="form-field form-field--inline">
            <input
              type="checkbox"
              data-testid="integration-dmr-toggle"
              checked={dmrEnabled}
              onChange={(e) => setDmrEnabled(e.target.checked)}
            />
            DMR referer hiding (<code className="code-inline">dmr=1</code>)
          </label>
          <label className="form-field form-field--inline">
            <input
              type="checkbox"
              data-testid="integration-utm-toggle"
              checked={utmEnabled}
              onChange={(e) => setUtmEnabled(e.target.checked)}
            />
            Append UTM
          </label>
        </div>
        {utmEnabled ? (
          <div className="stack" data-testid="integration-utm-fields">
            <label className="form-field" htmlFor="utm-source">
              utm_source
              <input
                id="utm-source"
                className="form-input form-input--sm"
                data-testid="integration-utm-source"
                value={utmSource}
                onChange={(e) => setUtmSource(e.target.value)}
              />
            </label>
            <label className="form-field" htmlFor="utm-medium">
              utm_medium
              <input
                id="utm-medium"
                className="form-input form-input--sm"
                data-testid="integration-utm-medium"
                value={utmMedium}
                onChange={(e) => setUtmMedium(e.target.value)}
              />
            </label>
            <label className="form-field" htmlFor="utm-campaign">
              utm_campaign
              <input
                id="utm-campaign"
                className="form-input form-input--sm"
                data-testid="integration-utm-campaign"
                placeholder={campaignId}
                value={utmCampaign}
                onChange={(e) => setUtmCampaign(e.target.value)}
              />
            </label>
          </div>
        ) : null}
        <IntegrationCopyRow label="Click URL" value={link} testId="integration-click-url" />
      </SectionCard>

      <SectionCard
        icon="download"
        title="Affiliate inbound S2S postback"
        desc="Give this URL to the affiliate network or offer partner. They POST JSON when a conversion settles — distinct from CAPI outbound to Meta/Google/TikTok."
      >
        <IntegrationCopyRow
          label="Postback URL"
          value={trackURL}
          testId="integration-inbound-url"
        />
        <IntegrationCopyRow
          label="JSON body template"
          value={inboundBody}
          testId="integration-inbound-body"
        />
        <IntegrationCopyRow
          label="Test with curl"
          value={inboundCurl}
          testId="integration-inbound-curl"
        />
        <p className="text-muted text-sm">
          Map network tokens to BidShard fields: click id → click_id, payout → payout_micro
          (micro-units) or omit and settle later. Requires Content-Length; chunked encoding is
          rejected on /track.
        </p>
      </SectionCard>

      <SectionCard
        icon="code"
        title="Zero-redirect (browser pixel)"
        desc="Fire /track from the landing page via fetch(). Set TRACK_CORS_ORIGINS on the tracker to include your LP origin."
      >
        <IntegrationCopyRow
          label="Landing page snippet"
          value={directSnippet}
          testId="integration-direct-snippet"
        />
        <p className="text-muted text-sm">
          Module loads bidshard-track.js; auto-picks fbclid/gclid/ttclid from the page query string.
        </p>
      </SectionCard>

      <SectionCard
        icon="eye"
        title="Impression / event examples"
        desc="Same POST /track URL as inbound S2S. Native JSON or protobuf (Content-Type application/x-protobuf)."
      >
        <IntegrationCopyRow
          label="Impression JSON example"
          value={impressionJSON}
          testId="integration-impression-json"
        />
        <p className="text-muted text-sm">
          Conversion body is in the Affiliate inbound S2S section above.
        </p>
      </SectionCard>

      <RtbTrackSection platform={platform} trackURL={trackURL} />

      <CampaignApplyTemplatesPanel
        campaignId={campaignId}
        canWrite={canWrite}
        trackingDomain={platform.trackingDomain}
      />

      {platform.edgeOpenRTB ? (
        <SectionCard
          icon="activity"
          title="OpenRTB exchange (SSP partners)"
          desc="Separate from /track RTB_MODE. SSPs POST OpenRTB 2.6 bid requests here — not the SDK /track path."
        >
          <IntegrationCopyRow
            label="Bid endpoint"
            value={openrtbURL}
            testId="integration-openrtb-url"
          />
          <p>
            <Link to="/rtb/integration" className="text-sm">
              RTB integration profile & validate-bid →
            </Link>
          </p>
        </SectionCard>
      ) : (
        <p className="text-muted text-sm">
          OpenRTB exchange: enable “Expose OpenRTB bid endpoint on edge” in Platform settings, or
          point partners at tracker :8181–8184/openrtb/bid.{' '}
          <Link to="/rtb/integration">Integration onboarding</Link>
        </p>
      )}

      <SectionCard
        icon="file-spreadsheet"
        title="Macro reference"
        desc="Substitute in landing URLs and partner postbacks."
      >
        <table className="data-table data-table--compact" data-testid="integration-macro-table">
          <thead>
            <tr>
              <th>Macro</th>
              <th>Meaning</th>
            </tr>
          </thead>
          <tbody>
            {(
              [
                ['{campaign_id}', 'Campaign UUID'],
                ['{click_id}', 'Unique click id (generated on redirect)'],
                ['{user_id}', 'Publisher user / visitor id'],
                ['{sub1}…{sub30}', 'Arbitrary sub-ids for source tracking'],
                ['{subid1}…{subid30}', 'Partner postback macro aliases'],
                [
                  'gclid, ttclid, fbclid',
                  'Attribution IDs — stored on event + forwarded to lander',
                ],
                ['ad_campaign_id', 'Network campaign id for Cost Sync join (often mirrors sub2)'],
              ] as const
            ).map(([macro, meaning]) => (
              <tr key={macro}>
                <td>
                  <code className="code-inline">{macro}</code>
                </td>
                <td>{meaning}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </SectionCard>
    </div>
  );
}
