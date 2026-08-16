import { useCallback, useEffect, useRef, useState } from 'react';
import { to } from '../../lib/to.js';
import { fetchTelegramFraud } from '../../helpers/tg_report_api.js';
import { useTelegramReport } from '../../hooks/use_telegram_report.js';
import { ErrorBlock } from '../../components/error_block.js';
import { TelegramReportShell } from './telegram_report_shell.js';

const PAGE_PATH = '/reports/telegram/fraud';

type FraudData = {
  blocked_clicks?: number;
  shadow_clicks?: number;
  freshness?: { stale?: boolean; lag_seconds?: number };
};

function StatTile({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="stat-tile">
      <div className="stat-tile__label">{label}</div>
      <div className="stat-tile__value">{value}</div>
    </div>
  );
}

export function TelegramFraudPage() {
  const tg = useTelegramReport(PAGE_PATH);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<FraudData | null>(null);
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
    const [res, err] = await to(fetchTelegramFraud(tg.reportParams));
    setLoading(false);
    if (err) {
      setError(err);
      setData(null);
      return;
    }
    setData((res as FraudData | null) ?? null);
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
      title="Telegram Fraud"
      pagePath={PAGE_PATH}
      freshness={data?.freshness ?? null}
      loading={loading}
      tg={tg}
      onSubmit={() => void load()}
    >
      {data ? (
        <div className="stats-row">
          <StatTile label="Blocked clicks" value={data.blocked_clicks ?? 0} />
          <StatTile label="Shadow clicks" value={data.shadow_clicks ?? 0} />
        </div>
      ) : (
        <p className="empty-hint">Perform a query to load report.</p>
      )}
    </TelegramReportShell>
  );
}
