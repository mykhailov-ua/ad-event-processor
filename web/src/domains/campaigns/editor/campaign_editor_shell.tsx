import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import type { ReactNode } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { PageLayout } from '@/shell/page_layout';
import { EDITOR_MAIN_COLUMN_CLASS } from '@/shell/filter_panel';
import { cn } from '@/lib/utils';
import { buildCampaignAuditHref } from '@/lib/audit_paths';
import { campaignReportPath } from '@/lib/campaign_nav';
import type { FlowPath } from '@/api/types';
import type { CampaignEditorFormState } from '@/domains/campaigns/editor/campaign_editor_types';
import { CAMPAIGN_EDITOR_MONO_EXTRALIGHT_CLASS } from '@/domains/campaigns/editor/campaign_click_query_limits';
import { adminTypography } from '@/lib/admin_kit';

export type CampaignEditorShellProps = {
  campaignId: string;
  campaignName: string;
  form: CampaignEditorFormState;
  flowPaths?: FlowPath[];
  saving?: boolean;
  clickUrl?: string;
  onFieldChange: <K extends keyof CampaignEditorFormState>(
    field: K,
    value: CampaignEditorFormState[K]
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
  const reportPath = campaignReportPath(campaignId);
  const paths = flowPaths.length > 0 ? flowPaths : [{ weight: 100, landers: [], offers: [] }];

  const pathsAside = (
    <section className="grid gap-4">
      <h2 className={cn('m-0', adminTypography.sectionTitle)}>Paths</h2>
      <div className="grid gap-4">
        {paths.map((path, pathIndex) => (
          <div className="grid gap-3" key={`path-${pathIndex}`}>
            <p className="m-0">
              <strong>Path {pathIndex + 1}</strong> / weight {path.weight ?? 100}
            </p>
            <div className="grid gap-2">
              <h3 className={cn('m-0', adminTypography.sectionTitle)}>Landers</h3>
              {(path.landers ?? []).length === 0 ? (
                <p className="m-0 text-muted-foreground">No landers</p>
              ) : (
                <ul className="m-0 flex list-disc flex-col gap-1 pl-5">
                  {(path.landers ?? []).map((lander, landerIndex) => (
                    <li key={`lander-${landerIndex}`}>
                      {lander.lander_id?.slice(0, 12) ?? `Lander ${landerIndex + 1}`} /{' '}
                      {lander.weight ?? 100}
                    </li>
                  ))}
                </ul>
              )}
            </div>
            <div className="grid gap-2">
              <h3 className={cn('m-0', adminTypography.sectionTitle)}>Offers</h3>
              {(path.offers ?? []).length === 0 ? (
                <p className="m-0 text-muted-foreground">No offers</p>
              ) : (
                <ul className="m-0 flex list-disc flex-col gap-1 pl-5">
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
      </div>
    </section>
  );

  return (
    <PageLayout
      aside={pathsAside}
      asideClassName="sticky top-0 max-h-full gap-4 border-l border-border p-4 lg:pl-6"
      description={`ID: ${campaignId}`}
      mainClassName="min-w-0"
      headerActions={
        <>
          <Button disabled={saving} loading={saving} type="button" onClick={onSave}>
            Save
          </Button>
          <Button type="button" variant="secondary" onClick={onClone}>
            Clone
          </Button>
          {reportPath ? (
            <Button asChild type="button" variant="secondary">
              <Link to={reportPath}>Report</Link>
            </Button>
          ) : null}
          <Button asChild type="button" variant="secondary">
            <Link to={buildCampaignAuditHref(campaignId)}>Audit log</Link>
          </Button>
          <Button type="button" variant="secondary" onClick={() => navigate('/campaigns')}>
            Close
          </Button>
        </>
      }
      title={campaignName || form.name || 'Campaign'}
    >
      {statusBanner}

      <section className={cn(EDITOR_MAIN_COLUMN_CLASS, 'gap-4')}>
        <h2 className={cn('m-0', adminTypography.sectionTitle)}>Main options</h2>

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
            <Label htmlFor="campaign-editor-url">Campaign URL</Label>
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
