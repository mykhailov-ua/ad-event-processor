import { Button } from '../system/button.js';
import styles from './reports_shared.module.css';

export type ReportFilterValues = {
  from: string;
  to: string;
  customerId: string;
  campaignId: string;
  limit: string;
  offset: string;
  cursor: string;
};

export type ReportFilterPanelProps = {
  values: ReportFilterValues;
  onChange: (field: keyof ReportFilterValues, value: string) => void;
  onApply: () => void;
};

export function ReportFilterPanel({ values, onChange, onApply }: ReportFilterPanelProps) {
  return (
    <section className={styles.filterGrid} aria-label="Report filters">
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>From</span>
        <input
          className={styles.textInput}
          type="datetime-local"
          value={values.from}
          onChange={(event) => onChange('from', event.target.value)}
        />
      </label>
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>To</span>
        <input
          className={styles.textInput}
          type="datetime-local"
          value={values.to}
          onChange={(event) => onChange('to', event.target.value)}
        />
      </label>
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>Customer ID</span>
        <input
          className={styles.textInput}
          type="text"
          value={values.customerId}
          onChange={(event) => onChange('customerId', event.target.value)}
          placeholder="UUID"
        />
      </label>
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>Campaign ID</span>
        <input
          className={styles.textInput}
          type="text"
          value={values.campaignId}
          onChange={(event) => onChange('campaignId', event.target.value)}
          placeholder="UUID"
        />
      </label>
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>Limit</span>
        <input
          className={styles.textInput}
          type="number"
          min={1}
          value={values.limit}
          onChange={(event) => onChange('limit', event.target.value)}
        />
      </label>
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>Offset</span>
        <input
          className={styles.textInput}
          type="number"
          min={0}
          value={values.offset}
          onChange={(event) => onChange('offset', event.target.value)}
        />
      </label>
      <label className={styles.filterField}>
        <span className={styles.filterLabel}>Cursor</span>
        <input
          className={styles.textInput}
          type="text"
          value={values.cursor}
          onChange={(event) => onChange('cursor', event.target.value)}
        />
      </label>
      <div className={styles.filterField}>
        <span className={styles.filterLabel}>&nbsp;</span>
        <Button type="button" variant="primary" onClick={onApply}>
          Apply filters
        </Button>
      </div>
    </section>
  );
}
