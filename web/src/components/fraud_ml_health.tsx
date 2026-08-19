import { useCallback, useEffect, useState } from 'react';
import { api } from '../helpers/api_client.js';
import {
  formatMlEvalAge,
  formatShadowPrecision,
  fraudProxyLabelDisclaimer,
  mlEvalBadgeStatus,
  mlEvalStatusLabel,
  type FraudMlHealthPayload,
} from '../helpers/fraud_ml_health.js';
import { StatusBadge } from './status_badge.js';

export type FraudMlHealthTileProps = {
  customerId?: string | null;
};


export function FraudMlHealthTile({ customerId }: FraudMlHealthTileProps) {
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<FraudMlHealthPayload | null>(null);

  const load = useCallback(async () => {
    if (!customerId) {
      setData(null);
      return;
    }
    setLoading(true);
    const qs = new URLSearchParams({ customer_id: customerId });
    const res = await api<FraudMlHealthPayload>(`/api/v1/dashboards/fraud?${qs.toString()}`);
    setData(res.data ?? null);
    setLoading(false);
  }, [customerId]);

  useEffect(() => {
    void load();
  }, [load]);

  if (!customerId) return null;

  const evalAge = formatMlEvalAge(data?.ml_eval_generated_at);
  const qs = `?customer_id=${encodeURIComponent(customerId)}`;

  return (
    <a
      href={`/dashboards/fraud${qs}`}
      className="metric-card metric-card--link"
      data-testid="fraud-kpi-ml-health"
    >
      <div className="metric-card__head">
        <div className="metric-card__label">ML protection</div>
        {data?.ml_eval_status ? (
          <StatusBadge
            status={mlEvalBadgeStatus(data.ml_eval_status)}
            label={mlEvalStatusLabel(data.ml_eval_status)}
          />
        ) : null}
      </div>
      <div className="metric-card__value font-mono text-sm">
        {loading ? '…' : formatShadowPrecision(data?.ml_precision)}
      </div>
      <p
        className={`text-xs ${evalAge.warning || data?.ml_eval_stale ? 'text-warning' : 'text-muted'}`}
      >
        {loading ? 'Loading eval…' : evalAge.label}
      </p>
    </a>
  );
}

export type FraudMlTrustPanelProps = {
  data: FraudMlHealthPayload;
};


export function FraudMlTrustPanel({ data }: FraudMlTrustPanelProps) {
  const evalAge = formatMlEvalAge(data.ml_eval_generated_at);
  const shards =
    data.ml_shards_consistent == null
      ? '—'
      : data.ml_shards_consistent
        ? 'All shards in sync'
        : 'Shard mismatch';

  return (
    <div className="stack stack--sm" data-testid="fraud-ml-trust-panel">
      <div className="button-row">
        <StatusBadge
          status={mlEvalBadgeStatus(data.ml_eval_status)}
          label={mlEvalStatusLabel(data.ml_eval_status)}
        />
        {data.ml_eval_stale ? <StatusBadge status="warning" label="stale eval" /> : null}
        {data.ml_drift_detected ? <StatusBadge status="warning" label="drift" /> : null}
      </div>
      <dl className="definition-list">
        <dt>ML active version</dt>
        <dd className="font-mono">{data.ml_active_version_id ?? '—'}</dd>
        <dt>Artifact hash</dt>
        <dd className="font-mono text-sm">
          {data.ml_artifact_hash ? `${data.ml_artifact_hash.slice(0, 12)}…` : '—'}
        </dd>
        <dt>Shadow precision (proxy)</dt>
        <dd>{formatShadowPrecision(data.ml_precision)}</dd>
        <dt>Shadow recall (proxy)</dt>
        <dd>{formatShadowPrecision(data.ml_recall)}</dd>
        <dt>Label method</dt>
        <dd className="font-mono">{data.ml_label_method ?? 'proxy'}</dd>
        <dt>Eval generated</dt>
        <dd className={evalAge.warning || data.ml_eval_stale ? 'text-warning' : undefined}>
          {data.ml_eval_generated_at ?? '—'}
          {evalAge.label ? ` (${evalAge.label})` : ''}
        </dd>
        <dt>Shard sync</dt>
        <dd>{shards}</dd>
        <dt>Drift</dt>
        <dd>
          {data.ml_drift_detected
            ? (data.ml_drift_summary ?? 'Traffic mix changed vs training.')
            : 'Within training band'}
        </dd>
      </dl>
      <p className="text-muted text-xs">{fraudProxyLabelDisclaimer()}</p>
    </div>
  );
}
