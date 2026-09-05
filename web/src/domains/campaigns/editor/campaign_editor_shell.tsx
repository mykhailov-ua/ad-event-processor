import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { ReactNode } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { PageLayout } from '@/shell/page_layout';
import type { FlowPath } from '@/api/types';
import type { CampaignEditorFormState } from '@/domains/campaigns/editor/campaign_editor_types';
import { CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS } from '@/domains/campaigns/editor/campaign_click_query_limits';

export type CampaignEditorShellProps = {
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
  onClone: () => void;
  advancedPanel?: ReactNode;
  statusBanner?: ReactNode;
};

export function CampaignEditorShell({
  campaignId,
  campaignName,
  form,
  flowPaths = [],
  saving = false,
  clickUrl,
  onFieldChange,
  onSave,
  onClone,
  advancedPanel,
  statusBanner,
}: CampaignEditorShellProps) {
  const navigate = useNavigate();
  const paths = flowPaths.length > 0 ? flowPaths : [{ weight: 100, landers: [], offers: [] }];

  const pathsAside = (
    <section className="flex flex-col gap-4">
      <h2 className="text-sm font-semibold text-foreground">Paths</h2>
      {paths.map((path, pathIndex) => (
        <div
          key={`path-${pathIndex}`}
          className="flex flex-col gap-3 border-t border-border pt-4 first:border-t-0 first:pt-0"
        >
          <p>
            <strong>Path {pathIndex + 1}</strong> / weight {path.weight ?? 100}
          </p>
          <div>
            <h3 className="text-sm font-semibold text-foreground">Landers</h3>
            {(path.landers ?? []).length === 0 ? (
              <p className="text-muted-foreground">No landers</p>
            ) : (
              <ul>
                {(path.landers ?? []).map((lander, landerIndex) => (
                  <li key={`lander-${landerIndex}`}>
                    {lander.lander_id?.slice(0, 12) ?? `Lander ${landerIndex + 1}`} /{' '}
                    {lander.weight ?? 100}
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div>
            <h3 className="text-sm font-semibold text-foreground">Offers</h3>
            {(path.offers ?? []).length === 0 ? (
              <p className="text-muted-foreground">No offers</p>
            ) : (
              <ul>
                {(path.offers ?? []).map((offer, offerIndex) => (
                  <li key={`offer-${offerIndex}`}>
                    {offer.offer_id?.slice(0, 20) ?? `Offer ${offerIndex + 1}`} /{' '}
                    {offer.weight ?? 100}
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      ))}
    </section>
  );

  return (
    <PageLayout
      aside={pathsAside}
      asideClassName="sticky top-0 max-h-full border-l border-border pl-4 pb-8 lg:pl-6"
      description={`ID: ${campaignId}`}
      mainClassName="min-w-0 gap-8 pb-8 pl-1 pr-2"
      workspaceClassName="min-h-0 flex-1 px-5 py-4"
      headerActions={
        <>
          <Button disabled={saving} loading={saving} type="button" onClick={onSave}>
            Save
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

      <section className="flex max-w-3xl flex-col gap-4">
        <h2 className="text-sm font-semibold text-foreground">Main options</h2>

        <div className="grid gap-4">
          <div className="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-x-4 gap-y-2">
            <Label htmlFor="campaign-editor-name">Name</Label>
            <Input
              disabled={saving}
              id="campaign-editor-name"
              value={form.name}
              onChange={(event) => onFieldChange('name', event.target.value)}
            />
          </div>

          <div className="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-x-4 gap-y-2">
            <Label htmlFor="campaign-editor-budget">Budget limit</Label>
            <Input
              disabled={saving}
              id="campaign-editor-budget"
              value={form.budget_limit}
              onChange={(event) => onFieldChange('budget_limit', event.target.value)}
            />
          </div>

          <div className="grid grid-cols-[8rem_minmax(0,1fr)] items-center gap-x-4 gap-y-2">
            <Label htmlFor="campaign-editor-status">Status</Label>
            <Input
              disabled={saving}
              id="campaign-editor-status"
              value={form.status}
              onChange={(event) => onFieldChange('status', event.target.value)}
            />
          </div>

          <div className="grid grid-cols-[8rem_minmax(0,1fr)] items-start gap-x-4 gap-y-2">
            <Label className="pt-2" htmlFor="campaign-editor-url">
              Campaign URL
            </Label>
            <Textarea
              className={CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS}
              id="campaign-editor-url"
              readOnly
              rows={3}
              value={clickUrl ?? `https://trk.example.com/click?campaign_id=${campaignId}`}
            />
          </div>
        </div>
      </section>

      {advancedPanel}
    </PageLayout>
  );
}
