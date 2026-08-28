import { useEffect, useRef, useState } from 'react';
import { useParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { boundCustomerId, hasBoundCustomer } from '../helpers/buyer_session.js';
import {
  probeStubReport,
  submitReportExport,
  pollReportJob,
  downloadReportExport,
  type StubProbeResult,
} from '../helpers/report_api.js';
import { reportTitle, retiredReportAlt, isRetiredReport } from '../models/report.js';
import { REPORT_DATE_PRESETS } from '../helpers/date_presets.js';
import { createInFlightGuard } from '../lib/async_guard.js';
import { Button, ButtonLink } from '../components/button.js';
import { PerfBlock } from '../components/perf_block.js';
import { StubBanner } from '../components/stub_banner.js';

const STUB_COPY: Record<string, { message: string; live: Array<{ href: string; label: string }> }> =
  {};

type ReportStubProbe =
  | StubProbeResult
  | {
      stub: boolean;
      ok: boolean;
      retired?: boolean;
      status?: number;
      message?: string;
    };

function stubCopy(reportKey: string) {
  if (STUB_COPY[reportKey]) return STUB_COPY[reportKey];
  return {
    message: `${reportTitle(reportKey)} is not available yet.`,
    live: [
      { href: '/reports/placements', label: 'Placements' },
      { href: '/reports/keywords', label: 'Keywords' },
    ],
  };
}

export function ReportStubPage() {
  const { reportKey = '' } = useParams();
  const title = reportTitle(reportKey);
  const copy = stubCopy(reportKey);
  const user = auth.getUser();
  const customerId = hasBoundCustomer(user?.role) ? boundCustomerId(user) : '';
  const exportGateRef = useRef(createInFlightGuard());

  const [loading, setLoading] = useState(true);
  const [probe, setProbe] = useState<ReportStubProbe | null>(null);
  const [exportStatus, setExportStatus] = useState<string | null>(null);
  const [exportLoading, setExportLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      if (isRetiredReport(reportKey)) {
        if (!cancelled) {
          setProbe({ stub: false, ok: false, retired: true });
          setLoading(false);
        }
        return;
      }
      const [result] = await to(probeStubReport(reportKey, customerId));
      if (!cancelled) {
        setProbe(result);
        setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [reportKey, customerId]);

  useEffect(() => () => exportGateRef.current.release(), []);

  const handleExport = async () => {
    if (!exportGateRef.current.tryAcquire()) return;
    if (!customerId) {
      setExportStatus('Select a customer to request an export.');
      exportGateRef.current.release();
      return;
    }
    setExportLoading(true);
    setExportStatus(null);
    const preset = REPORT_DATE_PRESETS[0];
    const result = await submitReportExport({
      customerId,
      reportKey,
      from: preset.from(),
      to: preset.to(),
    });
    setExportLoading(false);
    if (result.ok && result.jobId) {
      const polled = await pollReportJob(result.jobId);
      setExportStatus(
        polled.ok
          ? `Export ${polled.status}: downloading...`
          : `Export ${polled.status}: ${polled.message}`
      );
      if (polled.ok) {
        const [, dlErr] = await to(downloadReportExport(result.jobId, `${reportKey}.csv`));
        setExportStatus(
          dlErr
            ? `Export ready but download failed: ${dlErr.message}`
            : `Export downloaded: ${reportKey}.csv`
        );
      }
    } else {
      setExportStatus(
        result.stub
          ? `Export not available yet (${result.status}).`
          : `Job ${result.jobId ?? 'queued'}`
      );
    }
    exportGateRef.current.release();
  };

  const retiredAltLink = retiredReportAlt(reportKey);

  return (
    <div className="stack stack--lg" data-testid={`report-stub-${reportKey}`}>
      <div className="page-header">
        <div className="page-header__row">
          <h1 className="page-header__title">{title}</h1>
        </div>
        <p className="text-muted">
          <a href="/reports">{'<-'} Reports hub</a>
        </p>
      </div>

      {loading ? <p className="text-muted">Checking availability...</p> : null}

      {!loading && isRetiredReport(reportKey) ? (
        <>
          <StubBanner
            message={`${title} was retired. Use ${retiredAltLink?.label ?? 'a live report'} instead.`}
          />
          {retiredAltLink?.title ? (
            <p className="text-sm text-muted">{retiredAltLink.title}</p>
          ) : null}
        </>
      ) : null}

      {!loading && !isRetiredReport(reportKey) && probe?.stub ? (
        <>
          <StubBanner message={copy.message} />
          <p className="text-sm text-muted">This report is planned but not built yet.</p>
        </>
      ) : null}

      {!loading && probe && !probe.stub && !probe.ok ? (
        <p className="text-muted">
          {`Unexpected response (${probe.status}). ${probe.message || ''}`.trim()}
        </p>
      ) : null}

      <div className="section-card">
        <h2 className="subsection-title">Available now</h2>
        {isRetiredReport(reportKey) && retiredAltLink ? (
          <ButtonLink
            href={retiredAltLink.href}
            label={retiredAltLink.label}
            variant="primary"
            size="sm"
            title={retiredAltLink.title}
          />
        ) : (
          <div className="page-header__links">
            {copy.live.map((link, i) => (
              <span key={link.href}>
                {i > 0 ? <span className="text-muted"> , </span> : null}
                <a href={link.href}>{link.label}</a>
              </span>
            ))}
          </div>
        )}
      </div>

      {!isRetiredReport(reportKey) ? (
        <div className="toolbar-row">
          <Button
            label="Request CSV export"
            variant="secondary"
            loading={exportLoading}
            disabled={exportLoading || !customerId}
            onClick={() => void handleExport()}
          />
          {!customerId ? (
            <span className="text-sm text-muted">Customer context required for export.</span>
          ) : null}
        </div>
      ) : null}

      {exportStatus ? (
        <p id="stub-export-status" className="text-sm text-muted">
          {exportStatus}
        </p>
      ) : null}

      <PerfBlock id="report-stub-perf" />
    </div>
  );
}
