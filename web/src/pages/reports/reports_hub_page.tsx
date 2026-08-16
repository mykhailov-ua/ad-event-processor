import { useCallback, useEffect, useRef, useState } from 'react';
import { to } from '../../lib/to.js';
import * as auth from '../../helpers/auth.js';
import { boundCustomerId } from '../../helpers/buyer_session.js';
import {
  listSavedViews,
  savedViewHref,
  submitReportExport,
  pollReportJob,
  downloadReportExport,
  probeStubReport,
  type SavedViewRow,
} from '../../helpers/report_api.js';
import { REPORT_CATALOG, reportHref, reportIcon, retiredReportAlt } from '../../models/report.js';
import { parallelAll, isParallelSlotError } from '../../helpers/request_multiplex.js';
import { REPORT_DATE_PRESETS } from '../../helpers/date_presets.js';
import { createInFlightGuard } from '../../lib/async_guard.js';
import { Button, ButtonLink } from '../../components/button.js';
import { Icon } from '../../components/icon.js';
import { PerfBlock } from '../../components/perf_block.js';
import { StatusBadge } from '../../components/status_badge.js';

/**
 * Reports hub with catalog, saved views, and export.
 */
export function ReportsHubPage() {
  const user = auth.getUser();
  const customerId = boundCustomerId(user);
  const exportGateRef = useRef(createInFlightGuard());
  const exportAbortRef = useRef<AbortController | null>(null);

  const [savedViews, setSavedViews] = useState<SavedViewRow[]>([]);
  const [savedError, setSavedError] = useState<string | null>(null);
  const [exportStatus, setExportStatus] = useState<string | null>(null);
  const [exportLoading, setExportLoading] = useState(false);
  const [probing, setProbing] = useState(true);
  const [probeStub, setProbeStub] = useState<Record<string, boolean | null>>({});

  const loadProbes = useCallback(async () => {
    const keys = REPORT_CATALOG
      .filter((c) => !c.live && !c.retired)
      .map((c) => c.key);
    if (keys.length === 0) {
      setProbing(false);
      return;
    }
    const tasks = keys.map((key) => async () => {
      const [probe] = await to(probeStubReport(key, customerId ?? ''));
      return { key, stub: probe?.stub ?? true };
    });
    const [results] = await to(parallelAll(tasks, 3));
    setProbing(false);
    if (results) {
      const next: Record<string, boolean | null> = {};
      for (const slot of results) {
        if (isParallelSlotError(slot)) continue;
        next[slot.key] = slot.stub;
      }
      setProbeStub(next);
    }
  }, [customerId]);

  const loadSaved = useCallback(async () => {
    if (!customerId) return;
    const [views, err] = await to(listSavedViews(customerId));
    if (err) {
      setSavedError(err.message ?? 'Failed to load saved views');
      return;
    }
    setSavedViews((views ?? []) as SavedViewRow[]);
  }, [customerId]);

  useEffect(() => {
    void loadProbes();
    void loadSaved();
  }, [loadProbes, loadSaved]);

  const handleExport = async (reportKey: string) => {
    if (!customerId || !exportGateRef.current.tryAcquire()) return;
    exportAbortRef.current?.abort();
    const ctrl = new AbortController();
    exportAbortRef.current = ctrl;
    setExportLoading(true);
    setExportStatus(null);
    const preset = REPORT_DATE_PRESETS[0];
    const result = await submitReportExport({
      customerId,
      reportKey,
      from: preset.from(),
      to: preset.to(),
      signal: ctrl.signal,
    });
    setExportLoading(false);
    if (result.rateLimited) {
      setExportStatus(result.message);
      exportGateRef.current.release();
      return;
    }
    if (result.ok && result.jobId) {
      const polled = await pollReportJob(result.jobId, { signal: ctrl.signal });
      setExportStatus(polled.ok
        ? `Export ${polled.status}: downloading…`
        : `Export ${polled.status}: ${polled.message}`);
      if (polled.ok) {
        const [, dlErr] = await to(downloadReportExport(result.jobId, `${reportKey}.csv`));
        setExportStatus(dlErr
          ? `Export ready but download failed: ${dlErr.message}`
          : `Export downloaded: ${reportKey}.csv`);
      }
    } else {
      setExportStatus(result.stub
        ? `Export API not ready (${result.status}): ${result.message}`
        : `Export job: ${result.jobId ?? 'queued'}`);
    }
    exportGateRef.current.release();
  };

  useEffect(() => () => {
    exportAbortRef.current?.abort();
    exportGateRef.current.release();
  }, []);

  const liveCount = REPORT_CATALOG.filter((c) => c.live).length;

  return (
    <>
      <div className="page-header">
        <div className="page-header__row">
          <h1 className="page-header__title">Reports</h1>
          <span className="text-label-12 text-muted">
            {liveCount} live · {REPORT_CATALOG.length} total
          </span>
        </div>
        <p className="page-header__desc">
          Open a report or queue a CSV export for the selected customer.
        </p>
      </div>

      <section className="settings-panel" data-testid="reports-hub">
        <div className="settings-panel__header">
          <h2 className="settings-panel__title">Scheduled delivery</h2>
          <p className="settings-panel__desc">
            Email and webhook schedules are planned. Configure recipients in Settings when available.
          </p>
        </div>
      </section>

      <section className="settings-panel">
        <div className="settings-panel__header">
          <h2 className="settings-panel__title">Report catalog</h2>
          <p className="settings-panel__desc">
            Live reports are queryable today. Planned reports may return stub data until the API ships.
          </p>
        </div>
        <div className="settings-panel__body">
          <div className="table-wrapper table-wrapper--scroll">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">Report</th>
                  <th scope="col">Status</th>
                  <th scope="col" className="reports-hub__actions-cell">Actions</th>
                </tr>
              </thead>
              <tbody>
                {REPORT_CATALOG.map((card) => {
                  const retiredAlt = card.retired ? retiredReportAlt(card.key) : null;
                  const href = retiredAlt?.href ?? reportHref(card.key);
                  const isLive = card.live === true || probeStub[card.key] === false;
                  return (
                    <tr key={card.key} data-report-key={card.key}>
                      <td>
                        <a href={href} className="reports-hub__link">
                          <Icon name={reportIcon(card.key)} size={14} className="reports-hub__link-icon" />
                          {card.title}
                        </a>
                      </td>
                      <td>
                        {card.retired ? (
                          <StatusBadge status="archived" label="Retired" />
                        ) : isLive ? (
                          <StatusBadge status="ok" kind="service" label="Live" />
                        ) : probeStub[card.key] === true ? (
                          <StatusBadge status="planned" kind="service" label="Planned" />
                        ) : probing ? (
                          <StatusBadge status="pending" kind="invoice" label="Checking…" />
                        ) : (
                          <StatusBadge status="pending" kind="invoice" label="Checking…" />
                        )}
                      </td>
                      <td className="reports-hub__actions-cell">
                        <div className="reports-hub__actions">
                          <ButtonLink
                            href={href}
                            label={retiredAlt ? retiredAlt.label : 'Open'}
                            variant="secondary"
                            size="sm"
                          />
                          {!card.retired ? (
                            <Button
                              label="Export"
                              variant="ghost"
                              size="sm"
                              icon="download"
                              disabled={exportLoading || !customerId}
                              title={customerId ? 'Export CSV' : 'Customer context required'}
                              onClick={(e) => {
                                e.preventDefault();
                                void handleExport(card.key);
                              }}
                            />
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      </section>

      <section className="settings-panel">
        <div className="settings-panel__header">
          <h2 className="settings-panel__title">Saved presets</h2>
        </div>
        <div className="settings-panel__body">
          {!customerId ? (
            <div className="empty-state">
              <div className="empty-state__title">No customer context</div>
              <div className="empty-state__desc text-muted text-sm">
                Bind a customer in session to load saved report presets.
              </div>
            </div>
          ) : savedError ? (
            <p className="text-danger text-sm">{savedError}</p>
          ) : savedViews.length === 0 ? (
            <div className="empty-state">
              <div className="empty-state__title">No saved presets</div>
              <div className="empty-state__desc text-muted text-sm">
                Saved views from report pages will appear here.
              </div>
            </div>
          ) : (
            <div className="table-wrapper">
              <table className="data-table">
                <thead>
                  <tr>
                    <th scope="col">Preset</th>
                    <th scope="col">Report</th>
                    <th scope="col" className="reports-hub__actions-cell">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {savedViews.map((v, index) => (
                    <tr key={`${v.report_key ?? 'view'}-${index}`}>
                      <td><span className="text-label-14">{v.report_key ?? '—'}</span></td>
                      <td><span className="font-mono text-copy-13 text-muted">{v.report_key ?? '—'}</span></td>
                      <td className="reports-hub__actions-cell">
                        <ButtonLink
                          href={savedViewHref(v)}
                          label="Open"
                          variant="secondary"
                          size="sm"
                        />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </section>

      {exportStatus ? (
        <p className={`text-sm ${exportStatus.includes('failed') ? 'text-danger' : 'text-muted'}`}>
          {exportStatus}
        </p>
      ) : null}

      <PerfBlock id="reports-hub-perf" />
    </>
  );
}
