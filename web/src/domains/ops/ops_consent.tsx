import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

export type OpsConsentProps = {
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function OpsConsent({ payload, fetching, error, hasSnapshot }: OpsConsentProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Consent proofs">
        <OpsNav />
        {opsPanelError(error, 'Could not load consent proofs')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Consent proofs">
      <OpsNav />
      {payload ? (
        <JsonDashboardView payload={payload} />
      ) : (
        <p className="text-sm text-muted-foreground">No consent proof payload returned.</p>
      )}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
