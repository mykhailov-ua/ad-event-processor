import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import type { ReportSchedule } from '../../helpers/report_schedules_api.js';
import * as auth from '../../helpers/auth.js';
import { can, canReadCampaigns } from '../../helpers/permissions.js';
import { CustomerScopeBar } from '../integrations/customer_scope_bar.js';
import { Button } from '../system/button.js';
import { EmptyState } from '../system/empty_state.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageChrome } from '../system/page_chrome.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import { SettingsSubnav } from './settings_hub.js';
import styles from './settings_shared.module.css';

export type ReportSchedulesPanelProps = {
  customerId: string;
  schedules: ReportSchedule[];
  loading: boolean;
  error: unknown;
  busy: boolean;
  onCustomerApply: (customerId: string) => void;
  onCreate: (body: {
    customer_id: string;
    report_key: string;
    cron_expr: string;
    format?: string;
    enabled?: boolean;
  }) => void;
  onDelete: (id: string) => void;
};

export function ReportSchedulesPanel({
  customerId,
  schedules,
  loading,
  error,
  busy,
  onCustomerApply,
  onCreate,
  onDelete,
}: ReportSchedulesPanelProps) {
  const permissions = auth.getUser()?.permissions ?? [];
  const canList = canReadCampaigns(permissions);
  const canWrite = can(permissions, 'campaigns:write');

  const [reportKey, setReportKey] = useState('campaign-overview');
  const [cronExpr, setCronExpr] = useState('0 6 * * *');
  const [format, setFormat] = useState('csv');

  if (!canList) {
    return <ErrorBlock error={new Error('forbidden')} fallbackTitle="Report schedules access denied" />;
  }

  if (error) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load report schedules" />;
  }

  const onSubmit = (event: FormEvent) => {
    event.preventDefault();
    if (!customerId) return;
    onCreate({
      customer_id: customerId,
      report_key: reportKey.trim(),
      cron_expr: cronExpr.trim(),
      format: format.trim() || undefined,
      enabled: true,
    });
    setReportKey('campaign-overview');
    setCronExpr('0 6 * * *');
  };

  return (
    <div className={styles.root} data-testid="settings-report-schedules-page">
      <PageChrome
        title="Report schedules"
        badge={
          <Link to="/settings" className={styles.bannerLink}>
            Platform
          </Link>
        }
      />
      <SettingsSubnav />
      <CustomerScopeBar customerId={customerId} onApply={onCustomerApply} />

      {!customerId ? (
        <p className={styles.hint}>Enter a customer ID and apply to load schedules.</p>
      ) : (
        <>
          {canWrite ? (
            <form className={styles.formStack} onSubmit={onSubmit}>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Report key</span>
                <input
                  className={styles.textInput}
                  value={reportKey}
                  onChange={(e) => setReportKey(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Cron expression</span>
                <input
                  className={styles.textInput}
                  value={cronExpr}
                  onChange={(e) => setCronExpr(e.target.value)}
                />
              </label>
              <label className={styles.field}>
                <span className={styles.fieldLabel}>Format</span>
                <input
                  className={styles.textInput}
                  value={format}
                  onChange={(e) => setFormat(e.target.value)}
                />
              </label>
              <Button type="submit" variant="primary" disabled={busy || !reportKey.trim() || !cronExpr.trim()}>
                Create schedule
              </Button>
            </form>
          ) : null}

          <div className={styles.content}>
            {loading && schedules.length === 0 ? (
              <PageSkeleton rows={4} columns={6} />
            ) : schedules.length === 0 ? (
              <EmptyState message="No report schedules for this customer." />
            ) : (
              <div className={`${styles.gridTable} ${styles.colsSchedules}`} role="grid">
                <div className={styles.gridHeader} role="row">
                  <span className={styles.gridCell} role="columnheader">
                    Report
                  </span>
                  <span className={styles.gridCell} role="columnheader">
                    Cron
                  </span>
                  <span className={styles.gridCell} role="columnheader">
                    Next run
                  </span>
                  <span className={styles.gridCell} role="columnheader">
                    Format
                  </span>
                  <span className={styles.gridCell} role="columnheader">
                    On
                  </span>
                  <span className={styles.gridCell} role="columnheader">
                    Action
                  </span>
                </div>
                {schedules.map((row) => (
                  <div key={row.id ?? `${row.report_key}-${row.cron_expr}`} className={styles.gridRow} role="row">
                    <span className={styles.gridCell} role="gridcell">
                      {row.report_key ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.cron_expr ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.next_run_at ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.format ?? '-'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {row.enabled === false ? 'no' : 'yes'}
                    </span>
                    <span className={styles.gridCell} role="gridcell">
                      {canWrite && row.id ? (
                        <Button
                          type="button"
                         
                          variant="danger"
                          disabled={busy}
                          onClick={() => onDelete(row.id!)}
                        >
                          Delete
                        </Button>
                      ) : (
                        '-'
                      )}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  );
}
