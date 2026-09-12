import { adminSpacing, adminTypography } from '@/lib/admin_spacing';
import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { BillingForecast } from '@/api/types';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { CustomerTabShell } from '@/shell/customer_tab_shell';
import { StubBanner } from '@/shell/stub_banner';
import { displayMicro } from '@/lib/display';
export type CustomerDetailForecastTabProps = {
  forecast: BillingForecast | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

export function CustomerDetailForecastTab({
  forecast,
  fetching,
  error,
  hasSnapshot,
}: CustomerDetailForecastTabProps) {
  return (
    <CustomerTabShell
      blockingErrorOptions={{ unavailableTitle: 'Forecast unavailable' }}
      blockingErrorTitle="Could not load forecast"
      fetchState={{ fetching, error, hasSnapshot }}
    >
      {forecast?.ch_unavailable ? (
        <StubBanner
          message="ClickHouse impression history is unavailable. Ledger run-rate values are shown from Postgres; reload this tab to retry analytics."
          title="Forecast partially unavailable"
        />
      ) : null}
      {forecast ? (
        <Card>
          <CardHeader
            className={cn(
              'grid gap-2 sm:grid-cols-[1fr_auto] sm:items-center',
              adminSpacing.gap.md
            )}
          >
            <CardTitle>Billing forecast</CardTitle>
            <div className={adminSpacing.flex.buttonGroup}>
              {forecast.low_confidence ? <Badge variant="secondary">Low confidence</Badge> : null}
            </div>
          </CardHeader>
          <CardContent>
            <CustomerDetailPanel>
              <CustomerDetailRow label="Month" value={forecast.month} />
              <CustomerDetailRow
                label="Ledger MTD (micro)"
                value={displayMicro(forecast.ledger_mtd_micro)}
              />
              <CustomerDetailRow
                label="Run rate (micro/day)"
                value={displayMicro(forecast.ledger_run_rate_micro_per_day)}
              />
              <CustomerDetailRow
                label="Projected month end (micro)"
                value={displayMicro(forecast.projected_month_end_micro)}
              />
              <CustomerDetailRow label="Days remaining" value={forecast.days_remaining} />
            </CustomerDetailPanel>
          </CardContent>
        </Card>
      ) : null}
    </CustomerTabShell>
  );
}
