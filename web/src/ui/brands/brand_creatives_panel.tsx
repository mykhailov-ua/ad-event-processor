import { useEffect, useState, type FormEvent } from 'react';
import {
  createBrandCreative,
  fetchBrandCreatives,
  type BrandCreative,
} from '../../helpers/brands_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { to } from '../../lib/to.js';
import { Button } from '../system/button.js';
import styles from './brands_grid.module.css';

export type BrandCreativesPanelProps = {
  brandId: string;
  canWrite: boolean;
  onReloadBrands: () => void;
};

export function BrandCreativesPanel({ brandId, canWrite, onReloadBrands }: BrandCreativesPanelProps) {
  const [items, setItems] = useState<BrandCreative[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [name, setName] = useState('');
  const [landingUrl, setLandingUrl] = useState('');
  const [weight, setWeight] = useState('100');
  const [status, setStatus] = useState('active');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);
    void (async () => {
      const [result, err] = await to(fetchBrandCreatives(brandId, ctrl.signal));
      if (cancelled) return;
      if (err) {
        setError(err instanceof Error ? err.message : 'Failed to load creatives');
        setLoading(false);
        return;
      }
      setItems(result ?? []);
      setLoading(false);
    })();
    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [brandId]);

  const onCreate = async (event: FormEvent) => {
    event.preventDefault();
    const trimmedName = name.trim();
    const trimmedUrl = landingUrl.trim();
    const weightNum = Number.parseInt(weight, 10);
    if (!trimmedName || !trimmedUrl || !Number.isFinite(weightNum)) return;
    setBusy(true);
    try {
      await createBrandCreative(brandId, {
        name: trimmedName,
        landing_url: trimmedUrl,
        weight: weightNum,
        status: status.trim() || 'active',
      });
      pushToastMessage({ title: 'Creative created', message: trimmedName });
      setName('');
      setLandingUrl('');
      const [result] = await to(fetchBrandCreatives(brandId));
      setItems(result ?? []);
      onReloadBrands();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.creativesWrap}>
      <div className={styles.creativesGrid}>
        <div className={styles.creativesHeader}>
          <span>Name</span>
          <span>Landing URL</span>
          <span>Weight</span>
          <span>Status</span>
        </div>
        {loading ? <span className={styles.mutedCell}>Loading creatives...</span> : null}
        {error ? <span className={styles.mutedCell}>{error}</span> : null}
        {!loading &&
          items.map((creative) => (
            <div key={creative.id ?? creative.name} className={styles.creativesRow}>
              <span>{creative.name ?? '-'}</span>
              <span className={styles.mutedCell}>{creative.landing_url || '-'}</span>
              <span>{creative.weight != null ? String(creative.weight) : '-'}</span>
              <span>{creative.status ?? '-'}</span>
            </div>
          ))}
        {!loading && items.length === 0 ? (
          <span className={styles.mutedCell}>No creatives for this brand.</span>
        ) : null}
      </div>
      {canWrite ? (
        <form className={styles.createCreative} onSubmit={(event) => void onCreate(event)}>
          <input
            className={styles.textInput}
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder="Name"
            aria-label="Creative name"
            required
          />
          <input
            className={styles.textInput}
            value={landingUrl}
            onChange={(event) => setLandingUrl(event.target.value)}
            placeholder="https://"
            aria-label="Landing URL"
            required
          />
          <input
            className={styles.textInput}
            type="number"
            value={weight}
            onChange={(event) => setWeight(event.target.value)}
            aria-label="Weight"
          />
          <input
            className={styles.textInput}
            value={status}
            onChange={(event) => setStatus(event.target.value)}
            aria-label="Status"
          />
          <Button type="submit" variant="secondary" disabled={busy}>
            Add creative
          </Button>
        </form>
      ) : null}
    </div>
  );
}
