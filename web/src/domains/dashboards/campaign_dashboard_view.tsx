import { Link } from 'react-router-dom';

import { PageChrome } from '@/components/system/page_chrome';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { StubBanner } from '@/components/system/stub_banner';
import { Button } from '@/components/ui/button';
import { DatetimePicker } from '@/components/ui/datetime_picker';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { fromDatetimeLocalValue, toDatetimeLocalValue } from '@/lib/datetime_range';

export type CampaignDashboardViewProps = {
  campaignId: string;
  draftFrom: string;
  draftTo: string;
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  onDraftFromChange: (value: string) => void;
  onDraftToChange: (value: string) => void;
  onApply: () => void;
};

export function CampaignDashboardView({
  campaignId,
  draftFrom,
  draftTo,
  payload,
  fetching,
  error,
  hasSnapshot,
  licenseGated,
  onDraftFromChange,
  onDraftToChange,
  onApply,
}: CampaignDashboardViewProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (licenseGated) {
    return (
      <PageChrome title="Campaign dashboard">
        <StubBanner title="Dashboard unavailable" message="License or permission denied for campaign dashboard." />
      </PageChrome>
    );
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Campaign dashboard">
        <ErrorBlock title="Could not load campaign dashboard" message={error.message} />
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Campaign dashboard">
      <p className="text-sm text-muted-foreground">
        <Link className="hover:underline" to={`/campaigns/${campaignId}/edit`}>
          Back to campaign editor
        </Link>
        <span aria-hidden="true"> * </span>
        <span className="font-mono text-xs">{campaignId}</span>
      </p>

      <section className="ui-filter-panel sm:grid-cols-[1fr_1fr_auto] sm:items-end">
        <DatetimePicker
          id="campaign-dashboard-from"
          label="From"
          value={draftFrom}
          onChange={onDraftFromChange}
        />
        <DatetimePicker
          id="campaign-dashboard-to"
          label="To"
          value={draftTo}
          onChange={onDraftToChange}
        />
        <Button onClick={onApply} type="button">
          Apply range
        </Button>
      </section>

      {payload ? <JsonDashboardView payload={payload} /> : null}

      {error && hasSnapshot ? <ErrorBlock title="Refresh failed" message={error.message} /> : null}
    </PageChrome>
  );
}
