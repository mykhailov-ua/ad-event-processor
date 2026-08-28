import { useCallback, useEffect, useRef, useState } from 'react';
import type { DataFreshness, ReportEnvelope, ReportRow } from '../types/report.js';
import { to } from '../lib/to.js';
import { fetchTelegramFunnel } from '../helpers/tg_report_api.js';
import { useTelegramReport } from '../helpers/use_telegram_report.js';
import { ErrorBlock } from '../components/error_block.js';
import { TelegramReportEmpty, TelegramReportShell } from './report_telegram_shell.js';

const PAGE_PATH = '/reports/telegram/funnel';

export function TelegramFunnelPage() {
  const tg = useTelegramReport(PAGE_PATH);
  const [loading, setLoading] = useState(false);
  const [rows, setRows] = useState<ReportRow[] | null>(null);
  const [freshness, setFreshness] = useState<DataFreshness | null>(null);
  const [error, setError] = useState<unknown>(null);
  const initialLoadRef = useRef(false);

  const load = useCallback(async () => {
    const validationErr = tg.validateBeforeLoad();
    if (validationErr) {
      setError(new Error(validationErr));
      return;
    }
    setLoading(true);
    setError(null);
    const [res, err] = await to(fetchTelegramFunnel(tg.reportParams));
    setLoading(false);
    if (err) {
      setError(err);
      setRows(null);
      setFreshness(null);
      return;
    }
    const payload = (res as ReportEnvelope | null) ?? null;
    setRows(payload?.rows ?? []);
    setFreshness(payload?.freshness ?? null);
    tg.syncUrl();
  }, [tg]);

  useEffect(() => {
    if (initialLoadRef.current) return;
    initialLoadRef.current = true;
    void load();
  }, [load]);

  if (error) return <ErrorBlock error={error} />;

  return (
    <TelegramReportShell
      title="Telegram Funnel"
      pagePath={PAGE_PATH}
      freshness={freshness}
      loading={loading}
      tg={tg}
      onSubmit={() => void load()}
    >
      {rows && rows.length > 0 ? (
        <div className="table-wrapper table-section">
          <table className="data-table">
            <thead>
              <tr>
                <th scope="col">start_param</th>
                <th scope="col">Clicks</th>
                <th scope="col">Impressions</th>
                <th scope="col">Conversions</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={`${String(row.start_param)}-${index}`}>
                  <td>{String(row.start_param || '(empty)')}</td>
                  <td>{String(row.clicks ?? 0)}</td>
                  <td>{String(row.impressions ?? 0)}</td>
                  <td>{String(row.conversions ?? 0)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <TelegramReportEmpty />
      )}
    </TelegramReportShell>
  );
}
