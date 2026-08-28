import { useCallback, useEffect, useRef, useState } from 'react';
import { to } from '../lib/to.js';
import { fetchTelegramPremium } from '../helpers/tg_report_api.js';
import { useTelegramReport } from '../helpers/use_telegram_report.js';
import { ErrorBlock } from '../components/error_block.js';
import { TelegramReportShell } from './report_telegram_shell.js';

const PAGE_PATH = '/reports/telegram/premium';

type PremiumData = {
  premium_clicks?: number;
  non_premium_clicks?: number;
  premium_rate_pct?: number;
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

export function TelegramPremiumPage() {
  const tg = useTelegramReport(PAGE_PATH);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<PremiumData | null>(null);
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
    const [res, err] = await to(fetchTelegramPremium(tg.reportParams));
    setLoading(false);
    if (err) {
      setError(err);
      setData(null);
      return;
    }
    setData((res as PremiumData | null) ?? null);
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
      title="Telegram Premium"
      pagePath={PAGE_PATH}
      freshness={data?.freshness ?? null}
      loading={loading}
      tg={tg}
      onSubmit={() => void load()}
    >
      {data ? (
        <div className="stats-row">
          <StatTile label="Premium clicks" value={data.premium_clicks ?? 0} />
          <StatTile label="Non-premium clicks" value={data.non_premium_clicks ?? 0} />
          <StatTile label="Premium rate" value={`${(data.premium_rate_pct ?? 0).toFixed(1)}%`} />
        </div>
      ) : (
        <p className="empty-hint">Perform a query to load report.</p>
      )}
    </TelegramReportShell>
  );
}
