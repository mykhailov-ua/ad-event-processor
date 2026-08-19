import { useCallback, useEffect, useState } from 'react';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { ErrorBlock } from '../components/error_block.js';
import { StatusBadge } from '../components/status_badge.js';
import { Button } from '../components/button.js';
import {
  fetchOpsMLEvalReport,
  fetchOpsMLManualLabels,
  fetchOpsMLModelStatus,
  opsMlModelPollMs,
  truncateArtifactHash,
  type MLEvalMetricsBlock,
  type MLEvalReport,
  type MLFeatureImportance,
  type MLModelStatus,
  type MLModelVersion,
  type OpsMLManualLabel,
} from '../helpers/ops_ml_api.js';

type OpsMlTab = 'overview' | 'eval' | 'labels';


function formatTs(iso?: string): string {
  if (!iso) return '—';
  const parsed = new Date(iso);
  return Number.isNaN(parsed.getTime()) ? iso : parsed.toLocaleString();
}


function VersionSummary({ version, title }: { version?: MLModelVersion | null; title: string }) {
  if (!version) {
    return (
      <tr>
        <td>{title}</td>
        <td colSpan={4} className="text-muted">
          —
        </td>
      </tr>
    );
  }
  return (
    <tr>
      <td>{title}</td>
      <td className="font-mono text-sm">{version.id}</td>
      <td className="font-mono text-sm" title={version.artifact_hash}>
        {truncateArtifactHash(version.artifact_hash)}
      </td>
      <td>
        <StatusBadge status={version.status} kind="service" />
      </td>
      <td>{formatTs(version.created_at)}</td>
    </tr>
  );
}


function FeatureImportanceChart({ items }: { items: MLFeatureImportance[] }) {
  if (!items.length) {
    return <p className="text-muted text-sm">No feature importance in artifact metadata.</p>;
  }
  const max = Math.max(...items.map((item) => item.value), 0.0001);
  return (
    <div className="stack stack--sm" data-testid="ops-ml-importance-chart">
      {items.map((item) => (
        <div key={item.name} className="stack stack--xs">
          <div className="cluster cluster--sm">
            <span className="font-mono text-sm">{item.name}</span>
            <span className="text-muted text-xs ml-auto">{item.value.toFixed(4)}</span>
          </div>
          <div
            className="ops-ml-bar-track"
            style={{
              background: 'var(--color-border, #e5e7eb)',
              height: '6px',
              borderRadius: '3px',
              overflow: 'hidden',
            }}
            aria-hidden="true"
          >
            <div
              style={{
                background: 'var(--color-accent, #2563eb)',
                height: '100%',
                width: `${Math.min(100, (item.value / max) * 100)}%`,
              }}
            />
          </div>
        </div>
      ))}
    </div>
  );
}


function EvalMetricsBlock({
  title,
  block,
  disclaimer,
}: {
  title: string;
  block?: MLEvalMetricsBlock;
  disclaimer: string;
}) {
  const metrics = block ?? { status: 'empty', labeled_rows: 0 };
  return (
    <section className="stack stack--sm">
      <h3 className="subsection-title">{title}</h3>
      <p className="text-muted text-xs">{disclaimer}</p>
      <dl className="definition-list">
        <dt>Status</dt>
        <dd className="font-mono">{metrics.status}</dd>
        <dt>Label method</dt>
        <dd className="font-mono">{metrics.label_method ?? '—'}</dd>
        <dt>Labeled rows</dt>
        <dd className="font-mono">{metrics.labeled_rows ?? 0}</dd>
        {metrics.matched_rows != null ? (
          <>
            <dt>Matched rows</dt>
            <dd className="font-mono">{metrics.matched_rows}</dd>
          </>
        ) : null}
        {metrics.confidence ? (
          <>
            <dt>Confidence</dt>
            <dd className="font-mono">{metrics.confidence}</dd>
          </>
        ) : null}
        {metrics.precision != null ? (
          <>
            <dt>Precision</dt>
            <dd className="font-mono">{metrics.precision.toFixed(4)}</dd>
          </>
        ) : null}
        {metrics.recall != null ? (
          <>
            <dt>Recall</dt>
            <dd className="font-mono">{metrics.recall.toFixed(4)}</dd>
          </>
        ) : null}
      </dl>
    </section>
  );
}


export function OpsMlModelPage() {
  const [tab, setTab] = useState<OpsMlTab>('overview');
  const [status, setStatus] = useState<MLModelStatus | null>(null);
  const [evalReport, setEvalReport] = useState<MLEvalReport | null>(null);
  const [evalLoaded, setEvalLoaded] = useState(false);
  const [evalLoading, setEvalLoading] = useState(false);
  const [labels, setLabels] = useState<OpsMLManualLabel[]>([]);
  const [labelsLoaded, setLabelsLoaded] = useState(false);
  const [labelsLoading, setLabelsLoading] = useState(false);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);

  const loadStatus = useCallback(async () => {
    try {
      const data = await fetchOpsMLModelStatus();
      setStatus(data);
      setError(null);
    } catch (err) {
      setError(err);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadEval = useCallback(async () => {
    setEvalLoading(true);
    try {
      const report = await fetchOpsMLEvalReport();
      setEvalReport(report);
      setEvalLoaded(true);
    } catch (err) {
      setError(err);
    } finally {
      setEvalLoading(false);
    }
  }, []);

  const loadLabels = useCallback(async () => {
    setLabelsLoading(true);
    try {
      const rows = await fetchOpsMLManualLabels();
      setLabels(rows);
      setLabelsLoaded(true);
    } catch (err) {
      setError(err);
    } finally {
      setLabelsLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadStatus();
    const timer = window.setInterval(() => {
      void loadStatus();
    }, opsMlModelPollMs());
    return () => window.clearInterval(timer);
  }, [loadStatus]);

  useEffect(() => {
    if (tab !== 'labels' || labelsLoaded || labelsLoading) return;
    void loadLabels();
  }, [tab, labelsLoaded, labelsLoading, loadLabels]);

  useEffect(() => {
    if (tab !== 'eval' || evalLoaded || evalLoading) return;
    void loadEval();
  }, [tab, evalLoaded, evalLoading, loadEval]);

  if (loading && !status) {
    return <span className="text-muted">Loading…</span>;
  }
  if (error && !status) {
    return <ErrorBlock error={error} />;
  }

  const shardSync = status?.shard_sync ?? [];
  const importance = status?.importance ?? [];

  return (
    <div className="page stack" data-testid="ops-ml-model-page">
      <header className="page-header">
        <Breadcrumbs items={[{ label: 'Operations', href: '/ops' }, { label: 'ML model' }]} />
        <h1 className="page-title">ML model</h1>
        <p className="text-muted text-sm">
          Version lifecycle, Redis shard consistency, and shadow-eval metrics. Status refreshes
          every {opsMlModelPollMs() / 1000}s.
        </p>
      </header>

      <div className="cluster cluster--sm">
        <Button
          label="Overview"
          variant={tab === 'overview' ? 'primary' : 'secondary'}
          size="sm"
          onClick={() => setTab('overview')}
        />
        <Button
          label="Eval quality"
          variant={tab === 'eval' ? 'primary' : 'secondary'}
          size="sm"
          data-testid="ops-ml-eval-tab"
          onClick={() => setTab('eval')}
        />
        <Button
          label="Manual labels"
          variant={tab === 'labels' ? 'primary' : 'secondary'}
          size="sm"
          data-testid="ops-ml-labels-tab"
          onClick={() => setTab('labels')}
        />
      </div>

      {tab === 'overview' ? (
        <div className="stack stack--lg">
          <section className="section-card stack" data-testid="ops-ml-eval-panel">
            <h2 className="subsection-title">Shadow eval</h2>
            <dl className="definition-list">
              <dt>Drift</dt>
              <dd>
                <StatusBadge
                  status={status?.drift_detected ? 'failed' : 'ok'}
                  kind="service"
                  label={status?.drift_detected ? 'Drift detected' : 'No drift'}
                />
              </dd>
              <dt>Precision (proxy)</dt>
              <dd className="font-mono">
                {status?.precision != null ? status.precision.toFixed(4) : '—'}
              </dd>
              <dt>Recall (proxy)</dt>
              <dd className="font-mono">
                {status?.recall != null ? status.recall.toFixed(4) : '—'}
              </dd>
            </dl>
          </section>

          <section className="section-card stack" data-testid="ops-ml-versions-panel">
            <h2 className="subsection-title">Model versions</h2>
            <div className="table-wrapper">
              <table className="data-table">
                <thead>
                  <tr>
                    <th scope="col">Role</th>
                    <th scope="col">Version ID</th>
                    <th scope="col">Artifact hash</th>
                    <th scope="col">Status</th>
                    <th scope="col">Created</th>
                  </tr>
                </thead>
                <tbody>
                  <VersionSummary title="Active" version={status?.active_version} />
                  <VersionSummary title="Syncing" version={status?.syncing_version} />
                </tbody>
              </table>
            </div>
          </section>

          <section className="section-card stack" data-testid="ops-ml-redis-panel">
            <h2 className="subsection-title">Redis shard consistency</h2>
            <dl className="definition-list">
              <dt>Version on shards</dt>
              <dd className="font-mono">{status?.redis?.version_id || '—'}</dd>
              <dt>Hash</dt>
              <dd className="font-mono" title={status?.redis?.hash}>
                {truncateArtifactHash(status?.redis?.hash)}
              </dd>
              <dt>Applied at</dt>
              <dd>{formatTs(status?.redis?.applied_at)}</dd>
              <dt>Shards reporting</dt>
              <dd className="font-mono">{String(status?.redis?.shards_reporting ?? 0)}</dd>
              <dt>Consistent</dt>
              <dd>
                <StatusBadge
                  status={status?.redis?.shards_consistent ? 'ok' : 'warning'}
                  kind="service"
                  label={status?.redis?.shards_consistent ? 'Yes' : 'No'}
                />
              </dd>
            </dl>
          </section>

          <section className="section-card stack" data-testid="ops-ml-shard-sync-panel">
            <h2 className="subsection-title">Shard sync phases</h2>
            <div className="table-wrapper">
              <table className="data-table">
                <thead>
                  <tr>
                    <th scope="col">Shard</th>
                    <th scope="col">Model version</th>
                    <th scope="col">Phase</th>
                    <th scope="col">Started</th>
                  </tr>
                </thead>
                <tbody>
                  {shardSync.length === 0 ? (
                    <tr>
                      <td colSpan={4} className="text-muted">
                        No shard sync rows.
                      </td>
                    </tr>
                  ) : (
                    shardSync.map((row) => (
                      <tr key={`${row.shard_id}-${row.started_at}`}>
                        <td className="font-mono">{row.shard_id}</td>
                        <td className="font-mono text-sm">{row.model_version}</td>
                        <td>
                          <StatusBadge status={row.phase} kind="service" />
                        </td>
                        <td>{formatTs(row.started_at)}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </section>

          <section className="section-card stack" data-testid="ops-ml-importance-panel">
            <h2 className="subsection-title">Top feature importance</h2>
            <FeatureImportanceChart items={importance} />
          </section>
        </div>
      ) : tab === 'eval' ? (
        <section className="section-card stack" data-testid="ops-ml-eval-quality-panel">
          <h2 className="subsection-title">Eval quality</h2>
          {evalLoading ? <p className="text-muted text-sm">Loading eval report…</p> : null}
          {evalReport ? (
            <div className="stack stack--lg">
              <p className="text-muted text-xs">
                Generated {formatTs(evalReport.generated_at)} · window {evalReport.hours ?? '—'}h ·
                threshold {evalReport.threshold ?? '—'}
              </p>
              <EvalMetricsBlock
                title="Proxy metrics"
                block={evalReport.proxy_metrics}
                disclaimer="Proxy labels from ClickHouse heuristics — not accuracy or ground truth."
              />
              <EvalMetricsBlock
                title="Audited metrics"
                block={evalReport.audited_metrics}
                disclaimer="Human labels from ml_manual_labels; precision is not reported as accuracy."
              />
            </div>
          ) : !evalLoading ? (
            <p className="text-muted text-sm">Eval report unavailable.</p>
          ) : null}
        </section>
      ) : (
        <section className="section-card stack" data-testid="ops-ml-labels-panel">
          <h2 className="subsection-title">Manual labels</h2>
          {labelsLoading ? <p className="text-muted text-sm">Loading labels…</p> : null}
          <div className="table-wrapper">
            <table className="data-table">
              <thead>
                <tr>
                  <th scope="col">IP hash</th>
                  <th scope="col">Label</th>
                  <th scope="col">Reason</th>
                  <th scope="col">Source</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {!labelsLoading && labels.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="text-muted">
                      No manual labels.
                    </td>
                  </tr>
                ) : null}
                {labels.map((row) => (
                  <tr key={`${row.ip_hash}-${row.created_at ?? ''}`}>
                    <td className="font-mono text-sm">{row.ip_hash}</td>
                    <td className="font-mono">{row.label}</td>
                    <td>{row.reason || '—'}</td>
                    <td>{row.source || '—'}</td>
                    <td>{formatTs(row.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}
