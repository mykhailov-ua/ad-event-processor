import { useState, type FormEvent } from 'react';
import { createOffer, type Offer, windowRows } from '../../helpers/flows_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import styles from './panel.module.css';

export type OffersPanelProps = {
  items: Offer[];
  loading: boolean;
  canWrite: boolean;
  onReload: () => void;
};

function isHttpUrl(value: string): boolean {
  return /^https?:\/\//i.test(value);
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }, (_, index) => (
        <div key={`skel-${index}`} className={[styles.dataRow, styles.skeletonRow].join(' ')}>
          <span className={styles.bar} />
          <span className={styles.bar} />
          <span className={styles.bar} />
        </div>
      ))}
    </>
  );
}

export function OffersPanel({ items, loading, canWrite, onReload }: OffersPanelProps) {
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const { rows, truncated } = windowRows(items);

  const onCreate = async (event: FormEvent) => {
    event.preventDefault();
    const trimmedName = name.trim();
    const trimmedUrl = url.trim();
    if (!trimmedName || !trimmedUrl) return;
    if (!isHttpUrl(trimmedUrl)) {
      setCreateError('URL must start with http:// or https://');
      return;
    }
    setBusy(true);
    setCreateError(null);
    try {
      await createOffer({ name: trimmedName, url: trimmedUrl });
      pushToastMessage({ title: 'Offer created', message: trimmedName });
      setName('');
      setUrl('');
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setCreateError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.gridOffers} role="grid" aria-label="Offers">
      {canWrite ? (
        <form className={styles.createForm} onSubmit={(event) => void onCreate(event)}>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Name</span>
            <input
              className={styles.textInput}
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
              aria-label="Offer name"
            />
          </label>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>URL</span>
            <input
              className={styles.textInput}
              value={url}
              onChange={(event) => setUrl(event.target.value)}
              placeholder="https://"
              required
              aria-label="Offer URL"
            />
          </label>
          <Button type="submit" variant="primary" size="sm" disabled={busy}>
            Create offer
          </Button>
          {createError ? <span className={styles.uploadHint}>{createError}</span> : null}
        </form>
      ) : null}

      {truncated ? (
        <div className={styles.windowNote}>Showing first 500 offers.</div>
      ) : null}

      <div className={styles.headerRow} role="row">
        <div className={styles.headerCell} role="columnheader">
          Name
        </div>
        <div className={styles.headerCell} role="columnheader">
          URL
        </div>
        <div className={styles.headerCell} role="columnheader">
          ID
        </div>
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={styles.emptyWrap}>
          <EmptyState message="No offers yet." />
        </div>
      ) : null}

      {rows.map((offer) => {
        const id = offer.id ?? '';
        return (
          <div key={id} className={styles.dataRow} role="row">
            <div className={styles.nameCell} role="gridcell">
              {offer.name ?? id}
            </div>
            <div className={styles.mutedCell} role="gridcell">
              {offer.url ?? '-'}
            </div>
            <div className={styles.mutedCell} role="gridcell">
              {id}
            </div>
          </div>
        );
      })}
    </div>
  );
}
