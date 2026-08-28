import { useEffect, useState, type FormEvent } from 'react';
import type { RtbDeal, RtbDealCreateSpec } from '../../helpers/rtb_api.js';
import { Button } from '../system/button.js';
import styles from './deal_form_modal.module.css';

export type DealFormModalProps = {
  open: boolean;
  mode: 'create' | 'edit';
  initial?: RtbDeal | null;
  busy: boolean;
  error: string | null;
  onClose: () => void;
  onSubmit: (body: RtbDealCreateSpec) => void;
};

const EMPTY_FORM: RtbDealCreateSpec = {
  deal_id: '',
  customer_id: '',
  floor_micro: undefined,
  pacing: '',
  seats: undefined,
};

export function DealFormModal({
  open,
  mode,
  initial,
  busy,
  error,
  onClose,
  onSubmit,
}: DealFormModalProps) {
  const [form, setForm] = useState<RtbDealCreateSpec>(EMPTY_FORM);

  useEffect(() => {
    if (!open) return;
    if (mode === 'edit' && initial) {
      setForm({
        deal_id: initial.deal_id ?? '',
        customer_id: initial.customer_id ?? '',
        floor_micro: initial.floor_micro,
        pacing: initial.pacing ?? '',
        seats: initial.seats,
      });
      return;
    }
    setForm(EMPTY_FORM);
  }, [open, mode, initial]);

  if (!open) return null;

  const onFormSubmit = (event: FormEvent) => {
    event.preventDefault();
    if (!form.deal_id.trim() || !form.customer_id.trim()) return;
    onSubmit({
      deal_id: form.deal_id.trim(),
      customer_id: form.customer_id.trim(),
      floor_micro: form.floor_micro,
      pacing: form.pacing?.trim() || undefined,
      seats: form.seats,
    });
  };

  return (
    <div className={styles.backdrop} role="dialog" aria-modal="true" aria-labelledby="deal-form-title">
      <form className={styles.panel} onSubmit={onFormSubmit}>
        <div id="deal-form-title" className={styles.title}>
          {mode === 'create' ? 'Create RTB deal' : 'Edit RTB deal'}
        </div>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Deal ID</span>
          <input
            className={styles.textInput}
            value={form.deal_id}
            onChange={(event) => setForm((prev) => ({ ...prev, deal_id: event.target.value }))}
            required
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Customer ID</span>
          <input
            className={styles.textInput}
            value={form.customer_id}
            onChange={(event) => setForm((prev) => ({ ...prev, customer_id: event.target.value }))}
            required
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Floor (micro)</span>
          <input
            className={styles.textInput}
            type="number"
            value={form.floor_micro ?? ''}
            onChange={(event) => {
              const raw = event.target.value;
              setForm((prev) => ({
                ...prev,
                floor_micro: raw === '' ? undefined : Number.parseInt(raw, 10),
              }));
            }}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Pacing</span>
          <input
            className={styles.textInput}
            value={form.pacing ?? ''}
            onChange={(event) => setForm((prev) => ({ ...prev, pacing: event.target.value }))}
          />
        </label>
        <label className={styles.field}>
          <span className={styles.fieldLabel}>Seats</span>
          <input
            className={styles.textInput}
            type="number"
            value={form.seats ?? ''}
            onChange={(event) => {
              const raw = event.target.value;
              setForm((prev) => ({
                ...prev,
                seats: raw === '' ? undefined : Number.parseInt(raw, 10),
              }));
            }}
          />
        </label>
        {error ? <div className={styles.error}>{error}</div> : null}
        <div className={styles.actions}>
          <Button type="button" variant="secondary" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={busy}>
            {mode === 'create' ? 'Create' : 'Save'}
          </Button>
        </div>
      </form>
    </div>
  );
}
