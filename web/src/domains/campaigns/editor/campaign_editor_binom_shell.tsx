import { Button } from '@/components/ui/button';
import { useState, type ReactNode } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { PageLayout } from '@/shell/page_layout';
import type { FlowPath } from '@/api/types';
import type { CampaignEditorFormState } from '@/domains/campaigns/editor/campaign_editor';

export type CampaignEditorBinomShellProps = {
  campaignId: string;
  campaignName: string;
  form: CampaignEditorFormState;
  flowPaths?: FlowPath[];
  saving?: boolean;
  clickUrl?: string;
  onFieldChange: <K extends keyof CampaignEditorFormState>(
    field: K,
    value: CampaignEditorFormState[K],
  ) => void;
  onSave: () => void;
  onSaveAndClose: () => void;
  onClone: () => void;
  advancedPanel?: ReactNode;
  statusBanner?: ReactNode;
};

export function CampaignEditorBinomShell({
  campaignId,
  campaignName,
  form,
  flowPaths = [],
  saving = false,
  clickUrl,
  onFieldChange,
  onSave,
  onSaveAndClose,
  onClone,
  advancedPanel,
  statusBanner,
}: CampaignEditorBinomShellProps) {
  const navigate = useNavigate();
  const [advancedOpen, setAdvancedOpen] = useState(true);
  const paths = flowPaths.length > 0 ? flowPaths : [{ weight: 100, landers: [], offers: [] }];

  const pathsAside = (
    <section className="admin-panel admin-stack">
      <h2 className="admin-section-title">Paths</h2>
      {paths.map((path, pathIndex) => (
        <div key={`path-${pathIndex}`} className="admin-panel admin-stack">
          <p>
            <strong>Path {pathIndex + 1}</strong>  /  weight {path.weight ?? 100}
          </p>
          <div>
            <h3 className="admin-section-title">Landers</h3>
            {(path.landers ?? []).length === 0 ? (
              <p className="admin-muted">No landers</p>
            ) : (
              <ul>
                {(path.landers ?? []).map((lander, landerIndex) => (
                  <li key={`lander-${landerIndex}`}>
                    {lander.lander_id?.slice(0, 12) ?? `Lander ${landerIndex + 1}`}  / {' '}
                    {lander.weight ?? 100}
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div>
            <h3 className="admin-section-title">Offers</h3>
            {(path.offers ?? []).length === 0 ? (
              <p className="admin-muted">No offers</p>
            ) : (
              <ul>
                {(path.offers ?? []).map((offer, offerIndex) => (
                  <li key={`offer-${offerIndex}`}>
                    {offer.offer_id?.slice(0, 20) ?? `Offer ${offerIndex + 1}`}  / {' '}
                    {offer.weight ?? 100}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      ))}
      <Button type="button" variant="secondary">
        Add path
      </Button>
    </section>
  );

  return (
    <PageLayout
      aside={pathsAside}
      description={`ID: ${campaignId}`}
      headerActions={
        <>
          <Button disabled={saving} loading={saving} type="button" onClick={onSave}>
            Save
          </Button>
          <Button disabled={saving} loading={saving} type="button" variant="secondary" onClick={onSaveAndClose}>
            Save &amp; Close
          </Button>
          <Button type="button" variant="secondary" onClick={onClone}>
            Clone
          </Button>
          <Button asChild type="button" variant="secondary">
            <Link to={`/dashboards/campaign/${campaignId}`}>Report</Link>
          </Button>
          <Button type="button" variant="secondary" onClick={() => navigate('/campaigns')}>
            Close
          </Button>
        </>
      }
      title={campaignName || form.name || 'Campaign'}
    >
      {statusBanner}

      <section className="admin-panel admin-stack">
        <h2 className="admin-section-title">Main options</h2>

        <div className="admin-field-row">
          <label htmlFor="binom-campaign-name">Name</label>
          <input
            className="admin-input"
            disabled={saving}
            id="binom-campaign-name"
            value={form.name}
            onChange={(event) => onFieldChange('name', event.target.value)}
          />
        </div>

        <div className="admin-field-row">
          <label htmlFor="binom-traffic-template">Traffic source</label>
          <input
            className="admin-input"
            disabled={saving}
            id="binom-traffic-template"
            value={form.traffic_template_id}
            onChange={(event) => onFieldChange('traffic_template_id', event.target.value)}
          />
        </div>

        <div className="admin-field-row">
          <label htmlFor="binom-budget">Budget limit</label>
          <input
            className="admin-input"
            disabled={saving}
            id="binom-budget"
            value={form.budget_limit}
            onChange={(event) => onFieldChange('budget_limit', event.target.value)}
          />
        </div>

        <div className="admin-field-row">
          <label htmlFor="binom-status">Status</label>
          <input
            className="admin-input"
            disabled={saving}
            id="binom-status"
            value={form.status}
            onChange={(event) => onFieldChange('status', event.target.value)}
          />
        </div>

        <div className="admin-field-row">
          <label htmlFor="binom-flow">Flow ID</label>
          <input
            className="admin-input"
            disabled={saving}
            id="binom-flow"
            value={form.flow_id}
            onChange={(event) => onFieldChange('flow_id', event.target.value)}
          />
        </div>

        <div className="admin-field">
          <label htmlFor="binom-url">Campaign URL</label>
          <textarea
            className="admin-textarea"
            id="binom-url"
            readOnly
            rows={3}
            value={clickUrl ?? `https://trk.example.com/click?campaign_id=${campaignId}`}
          />
        </div>

        <Button type="button" variant="secondary" onClick={() => setAdvancedOpen((value) => !value)}>
          {advancedOpen ? 'Hide' : 'Show'} advanced settings
        </Button>

        {advancedOpen ? (
          <div className="admin-stack">
            <div className="admin-field-row">
              <label htmlFor="binom-postback">Postback %</label>
              <input className="admin-input" defaultValue="100" id="binom-postback" />
            </div>
            <div className="admin-field-row">
              <label htmlFor="binom-payout">Payout %</label>
              <input className="admin-input" defaultValue="100" id="binom-payout" />
            </div>
          </div>
        ) : null}
      </section>

      {advancedPanel ? <div className="admin-panel admin-stack">{advancedPanel}</div> : null}
    </PageLayout>
  );
}
