import { useCallback, useEffect, useState } from 'react';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';
import { fetchEdgeParityReport, type EdgeParityReport } from '../helpers/ops_edge_parity_api.js';
import { to } from '../lib/to.js';

const POLL_MS = 30_000;

function formatPct(value: number): string {
  return `${(value * 100).toFixed(2)}%`;
}

export function OpsEdgeParityPage() {
  const [report, setReport] = useState<EdgeParityReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const load = useCallback(async () => {
    const [data, err] = await to(fetchEdgeParityReport());
    if (err) {
      setError(err);
      setReport(null);
    } else {
      setError(null);
      setReport(data);
    }
    setLoading(false);
  }, []);

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => {
      void load();
    }, POLL_MS);
    return () => window.clearInterval(timer);
  }, [load]);

  if (loading && !report) {
    return <span className="text-muted">Loading...</span>;
  }
  if (error && !report) {
    return <ErrorBlock error={error} fallbackTitle="Failed to load edge parity report" />;
  }

  return (
    <div className="page stack" data-testid="ops-edge-parity-page">
      <header className="page-header">
        <Breadcrumbs items={[{ label: 'Operations', href: '/ops' }, { label: 'Edge parity' }]} />
        <h1 className="page-title">Edge parity</h1>
        <p className="text-muted text-sm">
          Compares edge ingress counters with tracker impression and click volume over the last 15
          minutes. Alert when divergence exceeds 5%.
        </p>
      </header>

      {report ? (
        <section className="section-card stack" data-testid="ops-edge-parity-panel">
          <div className="cluster cluster--sm">
            <h2 className="subsection-title">Current window</h2>
            <StatusBadge
              status={report.alert ? 'failed' : 'ok'}
              kind="service"
              label={report.alert ? 'Divergence alert' : 'Within threshold'}
            />
          </div>
          <dl className="definition-list">
            <dt>From</dt>
            <dd className="font-mono text-sm">{report.from}</dd>
            <dt>To</dt>
            <dd className="font-mono text-sm">{report.to}</dd>
            <dt>Edge ingress</dt>
            <dd className="font-mono">{report.edge_ingress}</dd>
            <dt>Tracker events</dt>
            <dd className="font-mono">{report.tracker_events}</dd>
            <dt>Divergence</dt>
            <dd className="font-mono">{formatPct(report.divergence_pct)}</dd>
            <dt>Edge blocked total</dt>
            <dd className="font-mono">{report.edge_blocked_total}</dd>
            <dt>Blacklist stale</dt>
            <dd className="font-mono">{report.blacklist_stale}</dd>
            {report.shard_mismatch_hint ? (
              <>
                <dt>Hint</dt>
                <dd className="font-mono text-sm">{report.shard_mismatch_hint}</dd>
              </>
            ) : null}
          </dl>
        </section>
      ) : (
        <p className="text-muted">No parity data returned.</p>
      )}
    </div>
  );
}
