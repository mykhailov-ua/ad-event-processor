import { useEffect, useState, type FormEvent } from 'react';
import { Select } from '../system/select.js';
import { Button } from '../system/button.js';
import styles from './billing_directory.module.css';

export type BillingFilterValues = {
  customer_id: string;
  status: string;
  month: string;
  min_total: string;
};

export type BillingFilterProps = {
  values: BillingFilterValues;
  onApply: (values: BillingFilterValues) => void;
};

const STATUS_OPTIONS = [
  { value: '', label: 'All statuses' },
  { value: 'open', label: 'Open' },
  { value: 'paid', label: 'Paid' },
  { value: 'void', label: 'Void' },
] as const;

export function BillingFilter({ values, onApply }: BillingFilterProps) {
  const [draft, setDraft] = useState<BillingFilterValues>(values);

  useEffect(() => {
    setDraft(values);
  }, [values]);

  return (
    <form
      className={styles.filters}
      onSubmit={(event: FormEvent) => {
        event.preventDefault();
        onApply(draft);
      }}
    >
      <div className={styles.filterRow}>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Customer ID</span>
          <input
            className={styles.textInput}
            value={draft.customer_id}
            onChange={(event) => setDraft((prev) => ({ ...prev, customer_id: event.target.value }))}
            placeholder="UUID"
            aria-label="Customer ID filter"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Status</span>
          <Select
            value={draft.status}
            onChange={(value) => setDraft((prev) => ({ ...prev, status: value }))}
            options={[...STATUS_OPTIONS]}
            aria-label="Invoice status filter"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Month</span>
          <input
            className={styles.textInput}
            value={draft.month}
            onChange={(event) => setDraft((prev) => ({ ...prev, month: event.target.value }))}
            placeholder="YYYY-MM"
            aria-label="Billing month filter"
          />
        </label>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Min total (micro)</span>
          <input
            className={styles.textInput}
            type="number"
            value={draft.min_total}
            onChange={(event) => setDraft((prev) => ({ ...prev, min_total: event.target.value }))}
            aria-label="Minimum total micro filter"
          />
        </label>
        <Button type="submit" variant="secondary" size="sm">
          Apply
        </Button>
      </div>
    </form>
  );
}
