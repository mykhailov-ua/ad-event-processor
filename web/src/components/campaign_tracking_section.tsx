import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { patchCampaign } from '../helpers/campaign_admin_api.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import type { OpsDoctorSummary } from '../types/ops.js';
import { defaultClickTemplate } from '../helpers/tracking_link.js';
import {
  buildInboundS2SBodyTemplate,
  buildInboundS2SCurl,
  buildTrackPostbackURL,
  trafficGuideSummary,
} from '../helpers/integration_kit.js';
import { buildDirectTrackSnippet } from '../static/track.js';
import { buildOpenRTBBidURL } from '../helpers/openrtb_endpoint.js';
import {
  TRAFFIC_SOURCE_TEMPLATES,
  templateParamMap,
  trafficSourceById,
} from '../models/traffic_source_templates.js';
import { buildTemplatedClickURL } from '../helpers/traffic_source_url.js';
import { costSyncHintsForNetwork } from '../helpers/cost_sync_url_hints.js';
import {
  ingressCostMacroPlaceholder,
  resolveIngressCostParam,
  type IngressCostParamName,
} from '../helpers/ingress_cost_url.js';
import type { IngressCostConfigDTO } from '../types/campaign.js';
import { SectionCard } from './section_card.js';
import { IntegrationCopyRow } from './integration_copy_row.js';
import { CampaignApplyTemplatesPanel } from './campaign_apply_templates_panel.js';
import { CampaignConversionMappingSection } from './campaign_conversion_mapping_section.js';
import { Button } from './button.js';

type ClickPresetSnapshot = {
  templateId: string;
  params: Record<string, string>;
};

function nonEmptyClickParams(params: Record<string, string>): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [key, value] of Object.entries(params)) {
    const trimmed = value.trim();
    if (trimmed) out[key] = trimmed;
  }
  return out;
}

function clickPresetEqual(a: ClickPresetSnapshot, b: ClickPresetSnapshot): boolean {
  if (a.templateId !== b.templateId) return false;
  const left = nonEmptyClickParams(a.params);
  const right = nonEmptyClickParams(b.params);
  const keys = new Set([...Object.keys(left), ...Object.keys(right)]);
  for (const key of keys) {
    if (left[key] !== right[key]) return false;
  }
  return true;
}

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
  ingressCostConfig?: IngressCostConfigDTO | null;
  trafficTemplateId?: string;
  clickQueryParams?: Record<string, string> | null;
};

function RtbTrackSection({ platform, trackURL }: { platform: PlatformState; trackURL: string }) {
  const mode = platform.rtbMode || 'off';
  const modeDesc: Record<string, string> = {
    off: 'In-process auction on /track is disabled. campaign_id must be present in the postback body.',
    shadow:
      'Auction runs before filters for metrics only - campaign_id in the body is unchanged. Use before promoting to live.',
    live: 'Auction may rewrite campaign_id to the winning line item before FilterEngine. No-bid rejects skip filters.',
  };
  const desc = modeDesc[mode] || modeDesc.off;

  return (
    <SectionCard
      icon="zap"
      title="In-app auction on /track"
      desc="Optional RTB_MODE=shadow|live - not a replacement for the OpenRTB exchange endpoint."
    >
      <p className="text-muted text-sm">
        SDK / single-endpoint flows only. Exchange partners use POST /openrtb/bid - see section
        below. Wire comparison is in `.cursor/rules/traffic.mdc` section 2.1.
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
  ingressCostConfig,
  trafficTemplateId,
  clickQueryParams,
}: CampaignTrackingSectionProps) {
  const configuredIngressParam = resolveIngressCostParam(ingressCostConfig?.param);
  const [params, setParams] = useState<Record<string, string>>({});
  const [selectedTemplateId, setSelectedTemplateId] = useState('direct-custom');
  const [trackingDomainOverride, setTrackingDomainOverride] = useState('');
  const [dmrEnabled, setDmrEnabled] = useState(false);
  const [utmEnabled, setUtmEnabled] = useState(false);
  const [utmSource, setUtmSource] = useState('');
  const [utmMedium, setUtmMedium] = useState('');
  const [utmCampaign, setUtmCampaign] = useState('');
  const [ingressCostEnabled, setIngressCostEnabled] = useState(Boolean(configuredIngressParam));
  const [ingressCostParam, setIngressCostParam] = useState<IngressCostParamName>(
    configuredIngressParam ?? 'cost'
  );
  const [platform, setPlatform] = useState<PlatformState>({
    clickTemplate: defaultClickTemplate(''),
    trackingDomain: '',
    edgeClick: false,
    edgeOpenRTB: false,
    rtbMode: 'off',
    rtbEnabled: false,
  });
  const [savedPreset, setSavedPreset] = useState<ClickPresetSnapshot | null>(null);
  const [presetBusy, setPresetBusy] = useState(false);
  const [presetError, setPresetError] = useState<string | null>(null);

  const currentPreset = useMemo(
    (): ClickPresetSnapshot => ({
      templateId: selectedTemplateId,
      params: { ...params },
    }),
    [selectedTemplateId, params]
  );

  const presetDirty =
    savedPreset != null && canWrite && !clickPresetEqual(currentPreset, savedPreset);

  const saveClickPreset = useCallback(async () => {
    if (!canWrite) return;
    setPresetBusy(true);
    setPresetError(null);
    const [, err] = await to(
      patchCampaign(campaignId, {
        traffic_template_id: selectedTemplateId,
        click_query_params: nonEmptyClickParams(params),
      })
    );
    setPresetBusy(false);
    if (err) {
      setPresetError(mapServiceError(err).message);
      return;
    }
    const snapshot: ClickPresetSnapshot = {
      templateId: selectedTemplateId,
      params: { ...params },
    };
    setSavedPreset(snapshot);
    pushToastMessage({ title: 'Click URL preset saved', message: 'Template and macros saved for this campaign.' });
  }, [canWrite, campaignId, params, selectedTemplateId]);

  const resetClickPreset = useCallback(() => {
    if (!savedPreset) return;
    setSelectedTemplateId(savedPreset.templateId);
    setParams({ ...savedPreset.params });
    setPresetError(null);
  }, [savedPreset]);

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

  useEffect(() => {
    const templateId = trafficTemplateId?.trim() || 'direct-custom';
    setSelectedTemplateId(templateId);
    const nextParams =
      clickQueryParams && Object.keys(clickQueryParams).length > 0 ? { ...clickQueryParams } : {};
    setParams((prev) => ({ ...prev, ...nextParams }));
    setSavedPreset({ templateId, params: { ...nextParams } });
    setPresetError(null);
  }, [campaignId, trafficTemplateId, clickQueryParams]);

  const selected = trafficSourceById(selectedTemplateId);
  const costSyncHints = costSyncHintsForNetwork(selected?.cost_sync);

  const clickTemplateBase =
    trackingDomainOverride.trim() || platform.clickTemplate || platform.trackingDomain;

  const link = useMemo(
    () =>
      buildTemplatedClickURL(clickTemplateBase, campaignId, params, {
        dmr: dmrEnabled,
        utm: utmEnabled
          ? {
              utm_source: utmSource || undefined,
              utm_medium: utmMedium || undefined,
              utm_campaign: utmCampaign || campaignId,
            }
          : undefined,
        ingressCost: ingressCostEnabled
          ? { param: ingressCostParam, value: ingressCostMacroPlaceholder(ingressCostParam) }
          : undefined,
      }),
    [
      clickTemplateBase,
      campaignId,
      params,
      dmrEnabled,
      utmEnabled,
      utmSource,
      utmMedium,
      utmCampaign,
      ingressCostEnabled,
      ingressCostParam,
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
          CAPI & Postbacks {'->'}
        </Link>
      </p>

      <details className="integration-guide" data-testid="traffic-guide">
        <summary>Traffic integration guide (buyer)</summary>
        <div className="stack text-sm">
          <p>
            Full wire contracts ship with the appliance as{' '}
            <code className="code-inline">.cursor/rules/traffic.mdc</code> (also in the operator
            bundle). Summary:
          </p>
          <ul className="list-plain">
            <li>Campaign traffic {'->'} Click URL (GET /click).</li>
            <li>
              Affiliate / CRM conversion {'->'} inbound S2S POST /track (JSON, Content-Length
              required).
            </li>
            <li>Lander pixel {'->'} zero-redirect fetch() to the same /track URL.</li>
            <li>
              Ad platforms (Meta/Google/TikTok) {'->'} configure on CAPI & Postbacks;
              ad-event-processor forwards after settlement.
            </li>
          </ul>
          <p className="text-muted">
            Edge tip: enable "Expose click URL on edge" in Platform settings so buyers hit :443
            instead of tracker ports.
          </p>
        </div>
      </details>

      <SectionCard
        icon="shield"
        title="Fallback landing vs brand landing"
        desc="Compliance fallback: suspicious traffic can redirect to an alternate URL while valid clicks reach brand landings."
      >
        <p className="text-muted text-sm" data-testid="integration-safe-page-hint">
          Brand landing: weighted brand creatives (Creative tab) or campaign target URL. Fallback
          landing: alternate URL for IVT / placement blacklist hits when enabled under
          Configuration.
        </p>
        <p>
          <Link to={`/campaigns/${campaignId}?tab=config`} className="text-sm">
            Configure compliance fallback {'->'}
          </Link>
        </p>
      </SectionCard>

      <div data-testid="campaign-url-builder">
        <SectionCard
          icon="copy"
          title="Get link"
          desc="Paste-ready click URL with traffic-source macros. Copy and drop into the ad network destination URL field."
        >
          <label className="form-field" htmlFor="url-builder-tracking-domain">
            Tracking domain or click template
            <input
              id="url-builder-tracking-domain"
              className="form-input form-input--sm"
              data-testid="url-builder-tracking-domain"
              placeholder={
                platform.clickTemplate ||
                platform.trackingDomain ||
                'https://track.example.com/click'
              }
              value={trackingDomainOverride}
              onChange={(e) => setTrackingDomainOverride(e.target.value)}
            />
          </label>
          <p className="text-muted text-sm">
            Leave blank to use platform default
            {platform.trackingDomain ? ` (${platform.trackingDomain})` : ''}. Campaign id is always{' '}
            <code className="code-inline">{campaignId}</code>.
          </p>
          <IntegrationCopyRow label="Click URL" value={link} testId="integration-click-url" />
        </SectionCard>
      </div>

      <SectionCard
        icon="globe"
        title="Traffic source macros"
        desc={
          platform.edgeClick
            ? 'Served on the edge (:8180/:443) with shard routing. Pick a template to pre-fill network macros.'
            : 'Enable "Expose click URL on edge" in Platform settings, or point traffic at tracker ports :8181-8184.'
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
        {costSyncHints ? (
          <div className="stack" data-testid="cost-sync-required-keys">
            <p className="text-sm">
              Cost Sync network:{' '}
              <Link to="/integrations/cost-sync" className="text-sm">
                {costSyncHints.label}
              </Link>{' '}
              (<code className="code-inline">{costSyncHints.apiNetworkId}</code>). Required query
              keys for spend join:
            </p>
            <table className="data-table data-table--compact">
              <thead>
                <tr>
                  <th>Query key</th>
                  <th>Role</th>
                </tr>
              </thead>
              <tbody>
                {costSyncHints.requiredKeys.map((row) => (
                  <tr key={row.key}>
                    <td>
                      <code className="code-inline">{row.key}</code>
                    </td>
                    <td>{row.hint}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            <p className="text-muted text-sm">
              Default token_mapping: {costSyncHints.tokenMappingDefault}. Override per credential on{' '}
              <Link to="/integrations/cost-sync" className="text-sm">
                Cost Sync
              </Link>
              .
            </p>
          </div>
        ) : null}
        {selected && selected.params.length > 0 ? (
          <details className="integration-guide" data-testid="template-param-reference" open>
            <summary>Template query keys ({selected.name})</summary>
            <table className="data-table data-table--compact mt-2">
              <thead>
                <tr>
                  <th>Key</th>
                  <th>Macro / value</th>
                  <th>Label</th>
                </tr>
              </thead>
              <tbody>
                {selected.params.map((param) => (
                  <tr key={param.key}>
                    <td>
                      <code className="code-inline">{param.key}</code>
                    </td>
                    <td>
                      <code className="code-inline">{param.value}</code>
                    </td>
                    <td>{param.label ?? '-'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </details>
        ) : null}
        {canWrite ? (
          <div className="button-row" data-testid="integration-preset-actions">
            <Button
              label={presetBusy ? 'Saving...' : 'Save macro preset'}
              variant="primary"
              size="sm"
              loading={presetBusy}
              disabled={presetBusy || !presetDirty}
              data-testid="integration-preset-save"
              onClick={() => void saveClickPreset()}
            />
            <Button
              label="Reset"
              variant="secondary"
              size="sm"
              disabled={presetBusy || !presetDirty}
              data-testid="integration-preset-reset"
              onClick={resetClickPreset}
            />
          </div>
        ) : null}
        {presetError ? (
          <p className="text-danger text-sm" data-testid="integration-preset-error">{presetError}</p>
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
          <summary>Sub 11-30</summary>
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
          <label className="form-field form-field--inline">
            <input
              type="checkbox"
              data-testid="integration-ingress-cost-toggle"
              checked={ingressCostEnabled}
              onChange={(e) => setIngressCostEnabled(e.target.checked)}
            />
            Append ingress cost macro
          </label>
        </div>
        {ingressCostEnabled ? (
          <label className="form-field" htmlFor="ingress-cost-param">
            Ingress cost query param
            <select
              id="ingress-cost-param"
              className="form-input form-input--sm"
              data-testid="integration-ingress-cost-param"
              value={ingressCostParam}
              onChange={(e) => setIngressCostParam(e.target.value as IngressCostParamName)}
            >
              <option value="cost">cost</option>
              <option value="cpc">cpc</option>
              <option value="bid">bid</option>
            </select>
          </label>
        ) : null}
        {ingressCostEnabled ? (
          <p className="text-muted text-sm" data-testid="integration-ingress-cost-hint">
            Optional spend on the click URL when the traffic source passes{' '}
            <code className="code-inline">{ingressCostMacroPlaceholder(ingressCostParam)}</code>.
            {configuredIngressParam ? (
              <>
                {' '}
                Campaign ingress_cost_config uses param{' '}
                <code className="code-inline">{configuredIngressParam}</code>
                {ingressCostConfig?.scale ? ` (${ingressCostConfig.scale} scale)` : ''}.
              </>
            ) : (
              <>
                {' '}
                Enable <code className="code-inline">ingress_cost_config</code> on the campaign to
                parse values into ClickHouse (Configuration tab).
              </>
            )}{' '}
            Does not replace Cost Sync API spend.
          </p>
        ) : null}
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
      </SectionCard>

      <SectionCard
        icon="download"
        title="Affiliate inbound S2S postback"
        desc="Give this URL to the affiliate network or offer partner. They POST JSON when a conversion settles - distinct from CAPI outbound to Meta/Google/TikTok."
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
          Map network tokens to ad-event-processor fields: click id {'->'} click_id, payout {'->'}
          payout_micro (micro-units) or omit and settle later. Requires Content-Length; chunked
          encoding is rejected on /track.
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
          Module loads <code>src/static/track.js</code>; auto-picks fbclid/gclid/ttclid from the
          page query string.
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

      <CampaignConversionMappingSection campaignId={campaignId} canWrite={canWrite} />

      {platform.edgeOpenRTB ? (
        <SectionCard
          icon="activity"
          title="OpenRTB exchange (SSP partners)"
          desc="Separate from /track RTB_MODE. SSPs POST OpenRTB 2.6 bid requests here - not the SDK /track path."
        >
          <IntegrationCopyRow
            label="Bid endpoint"
            value={openrtbURL}
            testId="integration-openrtb-url"
          />
          <p>
            <Link to="/rtb/integration" className="text-sm">
              RTB integration profile & validate-bid {'->'}
            </Link>
          </p>
        </SectionCard>
      ) : (
        <p className="text-muted text-sm">
          OpenRTB exchange: enable "Expose OpenRTB bid endpoint on edge" in Platform settings, or
          point partners at tracker :8181-8184/openrtb/bid.{' '}
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
                ['{sub1}...{sub30}', 'Arbitrary sub-ids for source tracking'],
                ['{subid1}...{subid30}', 'Partner postback macro aliases'],
                [
                  'gclid, ttclid, fbclid',
                  'Attribution IDs - stored on event + forwarded to lander',
                ],
                [
                  'status',
                  'Affiliate inbound status on conversion postback; mapped to payout in reports',
                ],
                ['ad_campaign_id', 'Network campaign id for Cost Sync join (often mirrors sub2)'],
                [
                  'cost, cpc, bid',
                  'Optional ingress spend macro when source passes spend on the click URL',
                ],
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
