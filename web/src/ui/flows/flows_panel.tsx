import { useMemo, useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import {
  createFlow,
  DEFAULT_FLOW_PATHS,
  summarizeFlowPaths,
  type Flow,
  windowRows,
} from '../../helpers/flows_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { formatLocaleDateTime } from '../../helpers/format_display.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import styles from './panel.module.css';

export type FlowsPanelProps = {
  items: Flow[];
  loading: boolean;
  canWrite: boolean;
  onReload: () => void;
};

function buildRowView(rows: Flow[]) {
  const len = rows.length;
  const ids = new Array<string>(len);
  const names = new Array<string>(len);
  const pathSummaries = new Array<string>(len);
  const createdLabels = new Array<string>(len);
  for (let i = 0; i < len; i += 1) {
    const flow = rows[i];
    const id = flow.id ?? '';
    ids[i] = id;
    names[i] = flow.name ?? id;
    pathSummaries[i] = summarizeFlowPaths(flow.paths);
    createdLabels[i] = formatLocaleDateTime(flow.created_at);
  }
  return { ids, names, pathSummaries, createdLabels, len };
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

export function FlowsPanel({ items, loading, canWrite, onReload }: FlowsPanelProps) {
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  const { rows, truncated } = windowRows(items);
  const rowView = useMemo(() => buildRowView(rows), [rows]);

  const onCreate = async (event: FormEvent) => {
    event.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) return;
    setBusy(true);
    setCreateError(null);
    try {
      await createFlow({ name: trimmedName, paths: DEFAULT_FLOW_PATHS });
      pushToastMessage({ title: 'Flow created', message: trimmedName });
      setName('');
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setCreateError(err instanceof Error ? err.message : 'Create failed');
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.grid} role="grid" aria-label="Flows">
      {canWrite ? (
        <form className={styles.createForm} onSubmit={(event) => void onCreate(event)}>
          <label className={styles.field}>
            <span className={styles.fieldLabel}>Name</span>
            <input
              className={styles.textInput}
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
              aria-label="Flow name"
            />
          </label>
          <Button type="submit" variant="primary" size="sm" disabled={busy}>
            Create flow
          </Button>
          {createError ? <span className={styles.uploadHint}>{createError}</span> : null}
        </form>
      ) : null}

      {truncated ? (
        <div className={styles.windowNote}>Showing first 500 flows.</div>
      ) : null}

      <div className={styles.headerRow} role="row">
        <div className={styles.headerCell} role="columnheader">
          Name
        </div>
        <div className={styles.headerCell} role="columnheader">
          Paths
        </div>
        <div className={styles.headerCell} role="columnheader">
          Created
        </div>
        <div className={styles.headerCell} role="columnheader">
          Actions
        </div>
      </div>

      {loading && items.length === 0 ? <SkeletonRows /> : null}

      {!loading && items.length === 0 ? (
        <div className={styles.emptyWrap}>
          <EmptyState message="No flows yet." />
        </div>
      ) : null}

      {Array.from({ length: rowView.len }, (_, index) => (
        <div key={rowView.ids[index]} className={styles.dataRow} role="row">
          <div className={styles.nameCell} role="gridcell">
            {rowView.names[index]}
          </div>
          <div className={styles.mutedCell} role="gridcell">
            {rowView.pathSummaries[index]}
          </div>
          <div className={styles.mutedCell} role="gridcell">
            {rowView.createdLabels[index]}
          </div>
          <div role="gridcell">
            <Link to={`/campaigns/flows/${rowView.ids[index]}/builder`}>Open builder</Link>
          </div>
        </div>
      ))}
    </div>
  );
}
