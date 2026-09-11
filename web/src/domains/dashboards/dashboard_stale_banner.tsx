import type { DashboardFreshness } from '@/domains/dashboards/dashboard_types';
import { StubBanner } from '@/shell/stub_banner';

type DashboardStaleBannerProps = {
  freshness: DashboardFreshness | undefined;
  sessionStaleBanner?: string;
};

export function DashboardStaleBanner({ freshness, sessionStaleBanner }: DashboardStaleBannerProps) {
  if (!freshness?.stale && !sessionStaleBanner) {
    return null;
  }

  const message =
    freshness?.freshness_label ??
    sessionStaleBanner ??
    'Analytics may be delayed while ClickHouse catches up.';

  return (
    <div data-testid="dashboard-stale-banner">
      <StubBanner
        message={`${message} Spend and counts may still reflect Postgres; revenue and ROI can lag.`}
        title="Stale analytics"
      />
    </div>
  );
}
