import { Button } from '@/components/ui/button';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsActionGroup,
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';

export type OpsRumProps = {
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onLoad: () => void;
};

export function OpsRum({ payload, fetching, error, hasSnapshot, onLoad }: OpsRumProps) {
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError error={error} pageTitle="RUM" title="Could not load RUM samples" />
    );
  }

  return (
    <OpsPageShell
      title="RUM"
      actions={
        <OpsActionGroup label="RUM">
          <Button disabled={fetching} loading={fetching} type="button" onClick={onLoad}>
            Load samples
          </Button>
        </OpsActionGroup>
      }
    >
      {payload ? (
        <JsonDashboardView payload={payload} />
      ) : (
        <p className="text-muted-foreground">Load RUM samples from the control plane.</p>
      )}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
