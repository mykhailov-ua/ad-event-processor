import type { BillingSummary, Invoice } from '../../helpers/billing_api.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { BillingFilter, type BillingFilterValues } from './billing_filter.js';
import { BillingGrid } from './billing_grid.js';
import { BillingSummaryStrip } from './billing_summary_strip.js';
import styles from './billing_directory.module.css';

export type BillingDirectoryProps = {
  items: Invoice[];
  total: number;
  limit: number;
  offset: number;
  filterValues: BillingFilterValues;
  summary: BillingSummary | null;
  loading: boolean;
  error: unknown;
  onFilterApply: (values: BillingFilterValues) => void;
  onOffsetChange: (offset: number) => void;
};

export function BillingDirectory({
  items,
  total,
  limit,
  offset,
  filterValues,
  summary,
  loading,
  error,
  onFilterApply,
  onOffsetChange,
}: BillingDirectoryProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load invoices" />;
  }

  return (
    <div className={styles.root}>
      <PageChrome title="Billing" badge={<LoadingCountBadge loading={loading} label={`${total} invoices`} />} />
      {summary ? <BillingSummaryStrip summary={summary} /> : null}
      <BillingFilter values={filterValues} onApply={onFilterApply} />
      <div className={styles.content}>
        <BillingGrid items={items} loading={loading} />
      </div>
      <div className={styles.footer}>
        <PaginationBar limit={limit} offset={offset} total={total} onOffsetChange={onOffsetChange} />
      </div>
    </div>
  );
}
