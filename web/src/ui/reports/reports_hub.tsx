import { Link } from 'react-router-dom';
import type { ReportCatalogRow } from '../../helpers/report_api.js';
import {
  isReportLive,
  reportHref,
  reportTitle,
  RETIRED_REPORT_ALTS,
} from '../../models/report.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { ReportSavedViewsPanel } from './report_saved_views_panel.js';
import type { SavedView } from '../../helpers/report_api.js';
import hubStyles from './reports_hub.module.css';
import styles from './reports_shared.module.css';

export type ReportsHubProps = {
  catalogRows: ReportCatalogRow[];
  savedViews: SavedView[];
  loading: boolean;
  savedViewsLoading: boolean;
  error: unknown;
  savedViewsError: unknown;
};

export function ReportsHub({
  catalogRows,
  savedViews,
  loading,
  savedViewsLoading,
  error,
  savedViewsError,
}: ReportsHubProps) {
  if (loading && catalogRows.length === 0 && !error) {
    return <PageSkeleton rows={4} columns={3} />;
  }

  if (error && catalogRows.length === 0) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load report catalog" />;
  }

  const retiredEntries = Object.entries(RETIRED_REPORT_ALTS);

  return (
    <div className={styles.root} data-testid="reports-hub-page">
      <PageChrome title="Reports" badge={<span>{catalogRows.length} catalog entries</span>} />
      <p className={hubStyles.intro}>
        Open a report to inspect ClickHouse-backed rows. Filters are driven by URL query parameters.
      </p>
      <ReportSavedViewsPanel
        views={savedViews}
        loading={savedViewsLoading}
        error={savedViewsError}
      />
      {retiredEntries.length > 0 ? (
        <section className={hubStyles.root}>
          <h2 className={hubStyles.sectionTitle}>Retired report aliases</h2>
          <div className={styles.cardGrid}>
            {retiredEntries.map(([key, href]) => (
              <div key={key} className={styles.retiredCard}>
                <p className={styles.cardTitle}>{reportTitle(key)}</p>
                <p className={styles.cardMeta}>Retired key: {key}</p>
                <Link className={styles.savedLink} to={href}>
                  Open replacement
                </Link>
              </div>
            ))}
          </div>
        </section>
      ) : null}
      <div className={styles.cardGrid} role="list">
        {catalogRows.map((row) => {
          const key = row.key ?? '';
          if (key === 'ghost-impression-funnel') return null;
          const live = isReportLive(key);
          const href = reportHref(key);
          const title = row.title ?? reportTitle(key);
          const description = row.description ?? '';
          if (!live) {
            return (
              <div key={key} className={styles.retiredCard} role="listitem">
                <p className={styles.cardTitle}>{title}</p>
                <p className={styles.cardDesc}>{description || 'No SPA route yet.'}</p>
                <p className={styles.cardMeta}>Key: {key}</p>
              </div>
            );
          }
          return (
            <Link key={key} to={href} className={styles.card} role="listitem">
              <h2 className={styles.cardTitle}>{title}</h2>
              {description ? <p className={styles.cardDesc}>{description}</p> : null}
              {row.category ? <p className={styles.cardMeta}>{row.category}</p> : null}
            </Link>
          );
        })}
      </div>
    </div>
  );
}
