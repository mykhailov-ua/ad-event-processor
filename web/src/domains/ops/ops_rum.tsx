import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Button } from '@/components/ui/button';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

export type OpsRumProps = {
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onLoad: () => void;
};

export function OpsRum({ payload, fetching, error, hasSnapshot, onLoad }: OpsRumProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="RUM">
        <OpsNav />
        {opsPanelError(error, 'Could not load RUM samples')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="RUM">
      <OpsNav />
      <Button disabled={fetching} onClick={onLoad} type="button" variant="outline">
        {fetching ? 'Loading...' : 'Load samples'}
      </Button>
      {payload ? (
        <JsonDashboardView payload={payload} />
      ) : (
        <p className="text-sm text-muted-foreground">Load RUM samples from the control plane.</p>
      )}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
