import type { BillingSummary } from '../../helpers/billing_api.js';
import { formatAmountMicro } from '../../helpers/money.js';
import styles from './billing_summary_strip.module.css';

export type BillingSummaryStripProps = {
  summary: BillingSummary;
};

export function BillingSummaryStrip({ summary }: BillingSummaryStripProps) {
  return (
    <div className={styles.root} role="region" aria-label="Billing summary">
      <div className={styles.card}>
        <span className={styles.label}>Invoiced MTD</span>
        <span className={styles.value}>{formatAmountMicro(summary.invoiced_mtd_micro)}</span>
      </div>
      <div className={styles.card}>
        <span className={styles.label}>Invoice count MTD</span>
        <span className={styles.value}>{String(summary.invoice_count_mtd ?? 0)}</span>
      </div>
      <div className={styles.card}>
        <span className={styles.label}>Undelivered notifications</span>
        <span className={styles.value}>
          {String(summary.undelivered_invoice_notifications ?? 0)}
        </span>
      </div>
    </div>
  );
}
