import { useEffect, useState, type FormEvent } from 'react';
import { Button } from '../system/button.js';
import styles from './brands_directory.module.css';

export type BrandsFilterProps = {
  customerId: string;
  onApply: (customerId: string) => void;
};

export function BrandsFilter({ customerId, onApply }: BrandsFilterProps) {
  const [draft, setDraft] = useState(customerId);

  useEffect(() => {
    setDraft(customerId);
  }, [customerId]);

  return (
    <form
      className={styles.filters}
      onSubmit={(event: FormEvent) => {
        event.preventDefault();
        onApply(draft.trim());
      }}
    >
      <div className={styles.filterRow}>
        <label className={styles.filterField}>
          <span className={styles.filterLabel}>Customer ID (required)</span>
          <input
            className={styles.textInput}
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
            placeholder="UUID"
            required
            aria-label="Customer ID"
          />
        </label>
        <Button type="submit" variant="secondary" size="sm">
          Apply
        </Button>
      </div>
    </form>
  );
}
