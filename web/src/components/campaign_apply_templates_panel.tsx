import { useState } from 'react';
import { to } from '../lib/to.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  applyCampaignTemplates,
  BUNDLED_AFFILIATE_TEMPLATES,
  BUNDLED_TRAFFIC_TEMPLATES,
} from '../helpers/integration_api.js';
import { SectionCard } from './section_card.js';
import { Button } from './button.js';

export type CampaignApplyTemplatesPanelProps = {
  campaignId: string;
  canWrite: boolean;
  trackingDomain?: string;
};

export function CampaignApplyTemplatesPanel({
  campaignId,
  canWrite,
  trackingDomain = '',
}: CampaignApplyTemplatesPanelProps) {
  const [trafficSource, setTrafficSource] = useState('');
  const [affiliateNetwork, setAffiliateNetwork] = useState('');
  const [busy, setBusy] = useState(false);

  const apply = async () => {
    if (!canWrite || busy) return;
    if (!trafficSource && !affiliateNetwork) {
      pushToastMessage({
        title: 'Nothing selected',
        message: 'Pick a traffic source and/or affiliate network template.',
      });
      return;
    }
    setBusy(true);
    const body: Record<string, string> = {};
    if (trafficSource) body.traffic_source = trafficSource;
    if (affiliateNetwork) body.affiliate_network = affiliateNetwork;
    if (trackingDomain.trim()) body.tracking_domain = trackingDomain.trim();

    const [result, err] = await to(applyCampaignTemplates(campaignId, body));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Apply failed', message: mapServiceError(err).message });
      return;
    }
    const parts: string[] = [];
    if (result?.traffic_source?.target_url) {
      parts.push(`target_url set`);
    }
    if (result?.affiliate_postback?.url_template) {
      parts.push(`postback configured`);
    }
    pushToastMessage({
      title: 'Templates applied',
      message: parts.length ? parts.join('; ') : 'Campaign integration schemas updated.',
    });
  };

  if (!canWrite) return null;

  return (
    <SectionCard
      icon="plug"
      title="Bundled integration templates"
      desc="Import presets under Integrations -> Schemas, then apply traffic and affiliate wiring to this campaign."
    >
      <div className="form-row">
        <label className="form-field" htmlFor="apply-traffic-source">
          Traffic source
          <select
            id="apply-traffic-source"
            className="form-input form-input--sm"
            value={trafficSource}
            data-testid="apply-traffic-source"
            onChange={(e) => setTrafficSource(e.target.value)}
          >
            <option value="">- none -</option>
            {BUNDLED_TRAFFIC_TEMPLATES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>
        <label className="form-field" htmlFor="apply-affiliate-network">
          Affiliate network
          <select
            id="apply-affiliate-network"
            className="form-input form-input--sm"
            value={affiliateNetwork}
            data-testid="apply-affiliate-network"
            onChange={(e) => setAffiliateNetwork(e.target.value)}
          >
            <option value="">- none -</option>
            {BUNDLED_AFFILIATE_TEMPLATES.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <div data-testid="apply-campaign-templates">
        <Button
          label={busy ? 'Applying...' : 'Apply templates to campaign'}
          variant="secondary"
          size="sm"
          loading={busy}
          disabled={busy}
          onClick={() => void apply()}
        />
      </div>
    </SectionCard>
  );
}
