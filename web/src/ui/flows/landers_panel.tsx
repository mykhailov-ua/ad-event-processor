import { useMemo, useState, type FormEvent, type ChangeEvent } from 'react';
import {
  createLander,
  type Lander,
  uploadLanderZip,
  windowRows,
} from '../../helpers/flows_api.js';
import { formatLocaleDateTime } from '../../helpers/format_display.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { Select } from '../system/select.js';
import styles from './panel.module.css';

export type LandersPanelProps = {
  items: Lander[];
  loading: boolean;
  canWrite: boolean;
  onReload: () => void;
};

const EMPTY_LANDER_OPTION = { value: '', label: 'Select lander...' } as const;

function buildRowView(rows: Lander[]) {
  const len = rows.length;
  const ids = new Array<string>(len);
  const names = new Array<string>(len);
  const urls = new Array<string>(len);
  const createdLabels = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const lander = rows[i];
    const id = lander.id ?? '';
    ids[i] = id;
    names[i] = lander.name ?? id;
    urls[i] = lander.hosted_url || lander.url || '-';
    createdLabels[i] = formatLocaleDateTime(lander.created_at);
  }
  return { ids, names, urls, createdLabels, len };
}

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }, (_, index) => (
        <div key={`skel-${index}`} className={[styles.dataRow, styles.skeletonRow].join(' ')}>
          <span className={styles.bar} />
          <span className={styles.bar} />
          <span className={styles.bar} />
          <span className={styles.bar} />
        </div>
      ))}
    </>
  );
}

export function LandersPanel({ items, loading, canWrite, onReload }: LandersPanelProps) {
  const [name, setName] = useState('');
  const [url, setUrl] = useState('');
  const [busy, setBusy] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState('');
  const [uploadBusy, setUploadBusy] = useState(false);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const { rows, truncated } = windowRows(items);
  const landerOptions = useMemo(
    () =>
      items
        .filter((item) => item.id)
        .map((item) => ({ value: item.id ?? '', label: item.name ?? item.id ?? '' })),
    [items]
  );
  const rowView = useMemo(() => buildRowView(rows), [rows]);

  const onCreate = async (event: FormEvent) => {
    event.preventDefault();
    if (!name.trim()) return;
    setBusy(true);
    setCreateError(null);
    try {
      await createLander({ name: name.trim(), url: url.trim() || undefined });
      pushToastMessage({ title: 'Lander created', message: name.trim() });
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

  const onUpload = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = '';
    if (!file || !selectedId) return;
    setUploadBusy(true);
    setUploadError(null);
    try {
      await uploadLanderZip(selectedId, file);
      pushToastMessage({ title: 'ZIP uploaded', message: file.name });
      onReload();
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed');
    } finally {
      setUploadBusy(false);
    }
  };

  return (
    <div className={styles.grid} role="grid" aria-label="Landers">
      {canWrite ? (
        <div className={styles.toolbar}>
          <form className={styles.createForm} onSubmit={(event) => void onCreate(event)}>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Name</span>
              <input
                className={styles.textInput}
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
                aria-label="Lander name"
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>URL</span>
              <input
                className={styles.textInput}
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                placeholder="https://"
                aria-label="Lander URL"
              />
            </label>
            <Button type="submit" variant="primary" size="sm" disabled={busy}>
              Create lander
            </Button>
            {createError ? <span className={styles.uploadHint}>{createError}</span> : null}
          </form>
          <div className={styles.createForm}>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>Upload ZIP to</span>
              <Select
                value={selectedId}
                onChange={setSelectedId}
                options={[EMPTY_LANDER_OPTION, ...landerOptions]}
                aria-label="Lander for ZIP upload"
              />
            </label>
            <label className={styles.field}>
              <span className={styles.fieldLabel}>ZIP file</span>
              <input
                type="file"
                accept=".zip,application/zip"
                disabled={!selectedId || uploadBusy}
                onChange={(event) => void onUpload(event)}
                aria-label="Hosted lander ZIP"
              />
            </label>
            {uploadError ? <span className={styles.uploadHint}>{uploadError}</span> : null}
            {!selectedId ? (
              <span className={styles.uploadHint}>Select a lander before uploading.</span>
            ) : null}
          </div>
        </div>
      ) : null}

      {truncated ? (
        <div className={styles.windowNote}>Showing first 500 landers. Narrow with server filters when available.</div>
      ) : null}

      <div className={styles.headerRow} role="row">
        <div className={styles.headerCell} role="columnheader">
          Name
        </div>
        <div className={styles.headerCell} role="columnheader">
          URL
        </div>
        <div className={styles.headerCell} role="columnheader">
          Created
        </div>
        <div className={styles.headerCell} role="columnheader">
          ID
        </div>
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={styles.emptyWrap}>
          <EmptyState message="No landers yet." />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => (
        <div key={rowView.ids[index]} className={styles.dataRow} role="row">
          <div className={styles.nameCell} role="gridcell">
            {rowView.names[index]}
          </div>
          <div className={styles.mutedCell} role="gridcell">
            {rowView.urls[index]}
          </div>
          <div className={styles.mutedCell} role="gridcell">
            {rowView.createdLabels[index]}
          </div>
          <div className={styles.mutedCell} role="gridcell">
            {rowView.ids[index]}
          </div>
        </div>
      ))}
    </div>
  );
}
