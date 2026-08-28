import { useEffect, useState, type FormEvent } from 'react';
import { Button } from '../system/button.js';
import styles from './brand_create_modal.module.css';

export type BrandCreateModalProps = {
  open: boolean;
  customerId: string;
  busy: boolean;
  error: string | null;
  onClose: () => void;
  onSubmit: (body: { customer_id: string; name: string }) => void;
};

export function BrandCreateModal({
  open,
  customerId,
  busy,
  error,
  onClose,
  onSubmit,
}: BrandCreateModalProps) {
  const [name, setName] = useState('');

  useEffect(() => {
    if (open) setName('');
  }, [open]);

  if (!open) return null;

  const onFormSubmit = (event: FormEvent) => {
    event.preventDefault();
    const trimmed = name.trim();
    if (!trimmed || !customerId) return;
    onSubmit({ customer_id: customerId, name: trimmed });
  };

  return (
    <div className={styles.backdrop} role="dialog" aria-modal="true" aria-labelledby="brand-create-title">
      <form className={styles.panel} onSubmit={onFormSubmit}>
        <div id="brand-create-title" className={styles.title}>
          Create brand
        </div>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Customer ID</span>
          <input className={styles.textInput} value={customerId} readOnly />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Name</span>
          <input
            className={styles.textInput}
            value={name}
            onChange={(event) => setName(event.target.value)}
            required
          />
        </label>
        {error ? <div className={styles.error}>{error}</div> : null}
        <div className={styles.actions}>
          <Button type="button" variant="secondary" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={busy || !customerId}>
            Create
          </Button>
        </div>
      </form>
    </div>
  );
}
