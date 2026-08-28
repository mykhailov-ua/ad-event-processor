import { Link } from 'react-router-dom';
import { reportHref } from '../../models/report.js';
import type { SavedView } from '../../helpers/report_api.js';
import styles from './reports_shared.module.css';

export type ReportSavedViewsPanelProps = {
  views: SavedView[];
  loading: boolean;
  error: unknown;
};

export function ReportSavedViewsPanel({ views, loading, error }: ReportSavedViewsPanelProps) {
  if (error || loading || views.length === 0) {
    return null;
  }

  return (
    <section className={styles.savedViews} aria-label="Saved views">
      <h2 className={styles.cardTitle}>Saved views</h2>
      <ul className={styles.savedList}>
        {views.map((view) => {
          const key = view.report_key ?? '';
          const href = key ? reportHref(key) : '/reports';
          return (
            <li key={view.id ?? `${key}-${view.name}`} className={styles.savedItem}>
              <span>{view.name ?? key}</span>
              <Link className={styles.savedLink} to={href}>
                Open
              </Link>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
