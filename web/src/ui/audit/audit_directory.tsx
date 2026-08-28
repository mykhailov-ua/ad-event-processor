import type { AuditLog } from '../../helpers/audit_api.js';
import { ErrorBlock } from '../system/error_block.js';
import { LoadingCountBadge } from '../system/loading_count_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { AuditExportBanner } from './audit_export_banner.js';
import { AuditFilter } from './audit_filter.js';
import { AuditGrid } from './audit_grid.js';
import { AuditToolbar } from './audit_toolbar.js';
import styles from './audit_directory.module.css';

export type AuditDirectoryProps = {
  items: AuditLog[];
  total: number;
  limit: number;
  offset: number;
  redactPii: boolean;
  loading: boolean;
  error: unknown;
  exporting: boolean;
  exportTruncated: boolean;
  exportNextCursor: string | null;
  onRedactPiiChange: (value: boolean) => void;
  onOffsetChange: (offset: number) => void;
  onExport: () => void;
  onContinueExport: () => void;
  onDismissExportBanner: () => void;
};

export function AuditDirectory({
  items,
  total,
  limit,
  offset,
  redactPii,
  loading,
  error,
  exporting,
  exportTruncated,
  exportNextCursor,
  onRedactPiiChange,
  onOffsetChange,
  onExport,
  onContinueExport,
  onDismissExportBanner,
}: AuditDirectoryProps) {
  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load audit log" />;
  }

  return (
    <div className={styles.root}>
      <PageChrome title="Audit log" badge={<LoadingCountBadge loading={loading} label={`${total} entries`} />} />
      <AuditToolbar exporting={exporting} onExport={onExport} />
      <AuditFilter redactPii={redactPii} onChange={onRedactPiiChange} />
      <AuditExportBanner
        truncated={exportTruncated}
        nextCursor={exportNextCursor}
        onContinue={exportNextCursor ? onContinueExport : undefined}
        onDismiss={onDismissExportBanner}
      />
      <div className={styles.content}>
        <AuditGrid items={items} loading={loading} />
      </div>
      <div className={styles.footer}>
        <PaginationBar limit={limit} offset={offset} total={total} onOffsetChange={onOffsetChange} />
      </div>
    </div>
  );
}
