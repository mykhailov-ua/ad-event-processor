import { useCallback, useEffect, useRef, useState } from 'react';
import { to } from '../../../lib/to.js';
import { fetchTelegramSummary } from '../../../helpers/tg_report_api.js';
import { useTelegramReport } from '../../hooks/use_telegram_report.js';
import { ErrorBlock } from '../../components/error_block.js';
import { TelegramReportShell } from './telegram_report_shell.js';

const PAGE_PATH = '/reports/telegram';

type SummaryData = {
  clicks?: number;
  impressions?: number;
  conversions?: number;
  premium?: number;
  motivated?: number;
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

function TelegramPieChart({ items }: { items: Array<{ label: string; value: number }> }) {
  const hostRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<{ destroy: () => void } | null>(null);

  useEffect(() => {
    if (!hostRef.current || items.length === 0) return;
    let cancelled = false;
    chartRef.current?.destroy();
    void import('../../../charts/pie_chart.js').then((mod) => {
      if (cancelled || !hostRef.current) return;
      chartRef.current = mod.mountPieChart(hostRef.current, items, 'Telegram attribution funnel');
    });
    return () => {
      cancelled = true;
      chartRef.current?.destroy();
      chartRef.current = null;
    };
  }, [items]);

  return <div ref={hostRef} />;
}

export function TelegramSummaryPage() {
  const tg = useTelegramReport(PAGE_PATH);
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<SummaryData | null>(null);
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
    const [res, err] = await to(fetchTelegramSummary(tg.reportParams));
    setLoading(false);
    if (err) {
      setError(err);
      setData(null);
      return;
    }
    setData((res as SummaryData | null) ?? null);
    tg.syncUrl();
  }, [tg]);

  useEffect(() => {
    if (initialLoadRef.current) return;
    initialLoadRef.current = true;
    void load();
  }, [load]);

  if (error) return <ErrorBlock error={error} />;

  const funnelItems = data
    ? [
        { label: 'Clicks', val: data.clicks ?? 0 },
        { label: 'Impressions', val: data.impressions ?? 0 },
        { label: 'Conversions', val: data.conversions ?? 0 },
        { label: 'Premium users', val: data.premium ?? 0 },
        { label: 'Motivated clicks', val: data.motivated ?? 0 },
      ]
    : [];

  const chartItems = data
    ? [
        { label: 'Clicks', value: data.clicks ?? 0 },
        { label: 'Impressions', value: data.impressions ?? 0 },
        { label: 'Conversions', value: data.conversions ?? 0 },
        { label: 'Premium', value: data.premium ?? 0 },
        { label: 'Motivated', value: data.motivated ?? 0 },
      ]
    : [];

  return (
    <TelegramReportShell
      title="Telegram Mini Apps"
      pagePath={PAGE_PATH}
      freshness={data?.freshness ?? null}
      loading={loading}
      tg={tg}
      onSubmit={() => void load()}
    >
      {data ? (
        <div className="stack stack--lg">
          <section>
            <h2 className="subsection-title">Attribution funnel</h2>
            <div className="funnel-grid">
              {funnelItems.map((item) => (
                <StatTile key={item.label} label={item.label} value={item.val} />
              ))}
            </div>
          </section>
          <section>
            <h2 className="subsection-title">Volume breakdown</h2>
            <TelegramPieChart items={chartItems} />
          </section>
        </div>
      ) : (
        <p className="empty-hint">Perform a query to load report.</p>
      )}
    </TelegramReportShell>
  );
}
