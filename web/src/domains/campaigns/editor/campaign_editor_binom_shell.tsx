import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { useState, type ReactNode } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { PageLayout } from '@/shell/page_layout';
import type { FlowPath } from '@/api/types';
import type { CampaignEditorFormState } from '@/domains/campaigns/editor/campaign_editor_types';

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
    <section className="rounded-md border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950 flex flex-col gap-3">
      <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Paths</h2>
      {paths.map((path, pathIndex) => (
        <div key={`path-${pathIndex}`} className="rounded-md border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950 flex flex-col gap-3">
          <p>
            <strong>Path {pathIndex + 1}</strong>  /  weight {path.weight ?? 100}
          </p>
          <div>
            <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Landers</h3>
            {(path.landers ?? []).length === 0 ? (
              <p className="text-zinc-500 dark:text-zinc-400">No landers</p>
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
            <h3 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Offers</h3>
            {(path.offers ?? []).length === 0 ? (
              <p className="text-zinc-500 dark:text-zinc-400">No offers</p>
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

      <section className="rounded-md border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950 flex flex-col gap-3">
        <h2 className="text-sm font-semibold text-zinc-900 dark:text-zinc-100">Main options</h2>

        <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
          <Label htmlFor="binom-campaign-name">Name</Label>
          <Input
            disabled={saving}
            id="binom-campaign-name"
            value={form.name}
            onChange={(event) => onFieldChange('name', event.target.value)}
          />
        </div>

        <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
          <Label htmlFor="binom-traffic-template">Traffic source</Label>
          <Input
            disabled={saving}
            id="binom-traffic-template"
            value={form.traffic_template_id}
            onChange={(event) => onFieldChange('traffic_template_id', event.target.value)}
          />
        </div>

        <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
          <Label htmlFor="binom-budget">Budget limit</Label>
          <Input
            disabled={saving}
            id="binom-budget"
            value={form.budget_limit}
            onChange={(event) => onFieldChange('budget_limit', event.target.value)}
          />
        </div>

        <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
          <Label htmlFor="binom-status">Status</Label>
          <Input
            disabled={saving}
            id="binom-status"
            value={form.status}
            onChange={(event) => onFieldChange('status', event.target.value)}
          />
        </div>

        <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
          <Label htmlFor="binom-flow">Flow ID</Label>
          <Input
            disabled={saving}
            id="binom-flow"
            value={form.flow_id}
            onChange={(event) => onFieldChange('flow_id', event.target.value)}
          />
        </div>

        <div className="grid gap-2">
          <Label htmlFor="binom-url">Campaign URL</Label>
          <Textarea
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
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
              <Label htmlFor="binom-postback">Postback %</Label>
              <Input defaultValue="100" id="binom-postback" />
            </div>
            <div className="grid grid-cols-[8rem_1fr] items-center gap-2">
              <Label htmlFor="binom-payout">Payout %</Label>
              <Input defaultValue="100" id="binom-payout" />
            </div>
          </div>
        ) : null}
      </section>

      {advancedPanel ? <div className="rounded-md border border-zinc-200 bg-white p-3 dark:border-zinc-800 dark:bg-zinc-950 flex flex-col gap-3">{advancedPanel}</div> : null}
    </PageLayout>
  );
}
