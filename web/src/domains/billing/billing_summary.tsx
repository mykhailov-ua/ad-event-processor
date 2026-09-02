import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { BillingSummary } from '@/api/types';
import { displayCount, displayMicro } from '@/lib/display';

export type BillingSummaryProps = {
  summary: BillingSummary | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

function KpiCard({ label, value }: { label: string; value: string }) {
  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{label}</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-2xl font-semibold tabular-nums">{value || '-'}</p>
      </CardContent>
    </Card>
  );
}

export function BillingSummarySection({
  summary,
  fetching,
  error,
  hasSnapshot,
}: BillingSummaryProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load billing summary" message={error.message} />;
  }

  return (
    <section className="grid gap-4">
      <h2 className="text-base font-semibold">Summary</h2>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <KpiCard
          label="Invoiced MTD (micro)"
          value={displayMicro(summary?.invoiced_mtd_micro, summary?.invoiced_mtd_display)}
        />
        <KpiCard
          label="Invoice count MTD"
          value={displayCount(summary?.invoice_count_mtd, summary?.invoice_count_mtd_display)}
        />
        <KpiCard
          label="Undelivered notifications"
          value={displayCount(
            summary?.undelivered_invoice_notifications,
            summary?.undelivered_invoice_notifications_display,
          )}
        />
        <KpiCard
          label="Customers with spend"
          value={displayCount(
            summary?.customers_with_spend_in_month,
            summary?.customers_with_spend_in_month_display,
          )}
        />
      </div>
    </section>
  );
}
