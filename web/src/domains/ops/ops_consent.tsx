import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { opsPanelError } from '@/domains/ops/ops_nav';
import {
  OpsPageBlockingError,
  OpsPageLoading,
  OpsPageShell,
} from '@/domains/ops/ops_page_shell';

export type OpsConsentProps = {
  payload: Record<string, unknown> | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function OpsConsent({ payload, fetching, error, hasSnapshot }: OpsConsentProps) {
  if (fetching && !hasSnapshot && !error) {
    return <OpsPageLoading />;
  }

  if (error && !hasSnapshot) {
    return (
      <OpsPageBlockingError
        error={error}
        pageTitle="Consent proofs"
        title="Could not load consent proofs"
      />
    );
  }

  return (
    <OpsPageShell title="Consent proofs">
      {payload ? (
        <JsonDashboardView payload={payload} />
      ) : (
        <p className="text-muted-foreground">No consent proof payload returned.</p>
      )}
      {error && hasSnapshot ? opsPanelError(error, 'Refresh failed') : null}
    </OpsPageShell>
  );
}
