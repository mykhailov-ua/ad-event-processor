import { useCallback, useMemo, useState, type CSSProperties } from 'react';
import { Link } from 'react-router-dom';
import {
  extractReportFreshness,
  extractReportRows,
  type ReportFetchResponse,
  type ReportJobSpec,
  submitReportExportJob,
  waitForReportJob,
  downloadReportJobExport,
  reportJobId,
  isReportJobFailed,
} from '../../helpers/report_api.js';
import { ApiError } from '../../helpers/api_client.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { formatReportColumnLabel, reportTitle } from '../../models/report.js';
import { ErrorBlock } from '../system/error_block.js';
import { FreshnessBadge } from '../system/freshness_badge.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { StubBanner } from '../system/stub_banner.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { PaginationBar } from '../system/pagination_bar.js';
import { ReportFilterPanel, type ReportFilterValues } from './report_filter_panel.js';
import { formatReportCellValue, reportRowKey } from '../../helpers/format_display.js';
import styles from './reports_shared.module.css';
import runnerStyles from './report_runner.module.css';

export type ReportRunnerProps = {
  reportKey: string;
  data: ReportFetchResponse | null;
  loading: boolean;
  error: unknown;
  filterValues: ReportFilterValues;
  onFilterChange: (field: keyof ReportFilterValues, value: string) => void;
  onFilterApply: () => void;
  onOffsetChange: (offset: number) => void;
  onReload: () => void;
};

export function ReportRunner({
  reportKey,
  data,
  loading,
  error,
  filterValues,
  onFilterChange,
  onFilterApply,
  onOffsetChange,
  onReload,
}: ReportRunnerProps) {
  const [exporting, setExporting] = useState(false);
  const title = reportTitle(reportKey);
  const rows = extractReportRows(data);
  const freshness = extractReportFreshness(data);
  const columns = useMemo(() => {
    if (rows.length === 0) return [];
    return Object.keys(rows[0]);
  }, [rows]);
  const columnTemplate = useMemo(
    () =>
      columns.length > 0
        ? columns.map(() => 'minmax(8rem, 1fr)').join(' ')
        : 'repeat(4, minmax(8rem, 1fr))',
    [columns]
  );
  const contentStyle = useMemo(
    () => ({ ['--report-cols']: columnTemplate }) as CSSProperties,
    [columnTemplate]
  );
  const rowKeys = useMemo(
    () => rows.map((row, rowIndex) => reportRowKey(row as Record<string, unknown>, rowIndex)),
    [rows]
  );
  const cellMatrix = useMemo(() => {
    if (rows.length === 0 || columns.length === 0) return [] as string[][];
    const matrix = new Array<string[]>(rows.length);
    for (let rowIndex = 0; rowIndex < rows.length; rowIndex += 1) {
      const row = rows[rowIndex];
      const cells = new Array<string>(columns.length);
      for (let colIndex = 0; colIndex < columns.length; colIndex += 1) {
        cells[colIndex] = formatReportCellValue(row[columns[colIndex]]);
      }
      matrix[rowIndex] = cells;
    }
    return matrix;
  }, [rows, columns]);

  const limit = Number.parseInt(filterValues.limit, 10);
  const offset = Number.parseInt(filterValues.offset, 10);
  const total = typeof data?.total === 'number' ? data.total : rows.length;
  const parsedLimit = Number.isFinite(limit) && limit > 0 ? limit : 50;
  const parsedOffset = Number.isFinite(offset) && offset >= 0 ? offset : 0;

  const stub = error instanceof ApiError && error.stub;

  const onExport = useCallback(async () => {
    if (!filterValues.customerId) {
      pushToastMessage({
        title: 'Export blocked',
        message: 'customer_id is required for report export jobs',
      });
      return;
    }
    setExporting(true);
    try {
      const spec: ReportJobSpec = {
        customer_id: filterValues.customerId,
        report_key: reportKey,
        from: filterValues.from || new Date(Date.now() - 7 * 86400000).toISOString(),
        to: filterValues.to || new Date().toISOString(),
        format: 'csv',
      };
      const accepted = await submitReportExportJob(spec);
      const jobId = reportJobId(accepted);
      if (!jobId) {
        throw new Error('Export job id missing from response');
      }
      pushToastMessage({ title: 'Export queued', message: `Job ${jobId}` });
      const finalStatus = await waitForReportJob(jobId);
      if (isReportJobFailed(finalStatus)) {
        throw new Error(finalStatus.error ?? 'Export job failed');
      }
      const ext = (finalStatus.format ?? 'csv').toLowerCase() === 'json' ? 'json' : 'csv';
      const filename = `${reportKey.replace(/\//g, '-')}.${ext}`;
      await downloadReportJobExport(jobId, filename);
      pushToastMessage({ title: 'Export ready', message: filename });
    } catch (err) {
      pushToastMessage({
        title: 'Export failed',
        message: err instanceof Error ? err.message : 'Export failed',
      });
    } finally {
      setExporting(false);
    }
  }, [filterValues, reportKey]);

  if (loading && !data && !error) {
    return <PageSkeleton rows={6} columns={5} />;
  }

  if (error && !data) {
    if (stub) {
      return (
        <div className={runnerStyles.root}>
          <PageChrome title={title} />
          <StubBanner
            title="Report not available"
            message={error instanceof Error ? error.message : 'This report endpoint returned 501.'}
          />
          <Link to="/reports">Back to reports hub</Link>
        </div>
      );
    }
    return <ErrorBlock error={error} fallbackTitle={`Failed to load ${title}`} onRetry={onReload} />;
  }

  return (
    <div className={styles.root} data-testid={`report-runner-${reportKey}`}>
      <PageChrome
        title={title}
        badge={
          freshness ? (
            <FreshnessBadge
              stale={Boolean(freshness.stale)}
              chLagSeconds={freshness.ch_lag_seconds ?? freshness.lag_seconds ?? 0}
            />
          ) : null
        }
      />
      <div className={styles.toolbar}>
        <Button type="button" variant="secondary" onClick={onReload} disabled={loading}>
          Refresh
        </Button>
        <Button type="button" variant="primary" onClick={() => void onExport()} disabled={exporting}>
          {exporting ? 'Exporting...' : 'Export CSV'}
        </Button>
        <Link to="/reports">All reports</Link>
      </div>
      <ReportFilterPanel values={filterValues} onChange={onFilterChange} onApply={onFilterApply} />
      <div className={styles.content} style={contentStyle}>
        {rows.length === 0 ? (
          <EmptyState message={loading ? 'Loading report rows...' : 'No rows for the selected filters.'} />
        ) : (
          <div className={styles.dataGrid} role="grid" aria-label={`${title} data`}>
            <div className={styles.gridHeader} role="row">
              {columns.map((column) => (
                <div key={column} className={styles.gridCell} role="columnheader">
                  {formatReportColumnLabel(column)}
                </div>
              ))}
            </div>
            {cellMatrix.map((cells, rowIndex) => (
              <div key={rowKeys[rowIndex]} className={styles.gridRow} role="row">
                {cells.map((cell, colIndex) => (
                  <div key={`${rowKeys[rowIndex]}-${columns[colIndex]}`} className={styles.gridCell} role="gridcell">
                    {cell}
                  </div>
                ))}
              </div>
            ))}
          </div>
        )}
      </div>
      <div className={styles.footer}>
        <PaginationBar
          limit={parsedLimit}
          offset={parsedOffset}
          total={total}
          onOffsetChange={onOffsetChange}
        />
      </div>
    </div>
  );
}
