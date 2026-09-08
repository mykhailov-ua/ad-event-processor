import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import type { BillingForecast } from '@/api/types';
import { CustomerDetailPanel } from '@/domains/customers/customer_detail_panel';
import { CustomerDetailRow } from '@/domains/customers/customer_detail_row';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
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
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load forecast" message={error.message} />;
  }

  if (!forecast) {
    return null;
  }

  return (
    <Card>
      <CardHeader className="grid grid-cols-[1fr_auto] items-center gap-2">
        <CardTitle className="text-base">Billing forecast</CardTitle>
        <div className="flex flex-wrap gap-2">
          {forecast.low_confidence ? <Badge variant="secondary">Low confidence</Badge> : null}
          {forecast.ch_unavailable ? <Badge variant="outline">ClickHouse unavailable</Badge> : null}
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
  );
}
