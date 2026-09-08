import { ErrorBlock } from '@/shell/error_block';
import { MetricCard } from '@/shell/metric_card';
import { PageSkeleton } from '@/shell/page_skeleton';
import type { BillingSummary } from '@/api/types';
import { billingPanelError } from '@/domains/billing/billing_nav';
import { displayCount, displayMicro } from '@/lib/display';

export type BillingSummaryProps = {
  summary: BillingSummary | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
};

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
    return billingPanelError(error, 'Could not load billing summary');
  }

  return (
    <section className="grid gap-4">
      <h2 className="text-base font-semibold">Summary</h2>
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          accent={1}
          label="Invoiced MTD (micro)"
          value={displayMicro(summary?.invoiced_mtd_micro, summary?.invoiced_mtd_display) || '-'}
        />
        <MetricCard
          accent={4}
          label="Invoice count MTD"
          value={displayCount(summary?.invoice_count_mtd, summary?.invoice_count_mtd_display) || '-'}
        />
        <MetricCard
          accent={3}
          label="Undelivered notifications"
          value={
            displayCount(
              summary?.undelivered_invoice_notifications,
              summary?.undelivered_invoice_notifications_display
            ) || '-'
          }
        />
        <MetricCard
          accent={2}
          label="Customers with spend"
          value={
            displayCount(
              summary?.customers_with_spend_in_month,
              summary?.customers_with_spend_in_month_display
            ) || '-'
          }
        />
      </div>
    </section>
  );
}
