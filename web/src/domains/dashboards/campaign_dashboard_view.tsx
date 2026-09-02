import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { StubBanner } from '@/shell/stub_banner';
import type { Flow } from '@/api/types';
import { CampaignReportDirectory } from '@/domains/campaigns/report/campaign_report_directory';

export type CampaignDashboardViewProps = {
  campaignId: string;
  campaignName?: string;
  payload: Record<string, unknown> | undefined;
  flow?: Flow;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  licenseGated: boolean;
  draftQ: string;
  onDraftQChange: (value: string) => void;
  onRefresh: () => void;
};

export function CampaignDashboardView({
  campaignId,
  campaignName,
  payload,
  flow,
  fetching,
  error,
  hasSnapshot,
  licenseGated,
  draftQ,
  onDraftQChange,
  onRefresh,
}: CampaignDashboardViewProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (licenseGated) {
    return (
      <div className="admin-page">
        <StubBanner title="Dashboard unavailable" message="License or permission denied for campaign dashboard." />
      </div>
    );
  }

  if (error && !hasSnapshot) {
    return (
      <div className="admin-page">
        <ErrorBlock title="Could not load campaign dashboard" message={error.message} />
      </div>
    );
  }

  return (
    <CampaignReportDirectory
      campaignId={campaignId}
      campaignName={campaignName}
      draftQ={draftQ}
      fetching={fetching}
      flow={flow}
      payload={payload}
      onDraftQChange={onDraftQChange}
      onRefresh={onRefresh}
    />
  );
}
