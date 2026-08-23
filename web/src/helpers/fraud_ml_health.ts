export type MlEvalStatus = 'healthy' | 'drift_detected' | 'eval_stale' | 'eval_unavailable';

export type FraudMlHealthPayload = {
  ml_active_version_id?: string;
  ml_artifact_hash?: string;
  ml_precision?: number;
  ml_recall?: number;
  ml_drift_detected?: boolean;
  ml_drift_summary?: string;
  ml_eval_generated_at?: string;
  ml_eval_status?: MlEvalStatus;
  ml_eval_stale?: boolean;
  ml_label_method?: string;
  ml_shards_consistent?: boolean | null;
};

const PROXY_LABEL_DISCLAIMER =
  'Shadow precision is estimated on proxy labels, not human-audited ground truth.';

export function fraudProxyLabelDisclaimer(): string {
  return PROXY_LABEL_DISCLAIMER;
}

export function mlEvalBadgeStatus(status?: MlEvalStatus): 'ok' | 'warning' | 'failed' | 'pending' {
  switch (status) {
    case 'healthy':
      return 'ok';
    case 'drift_detected':
    case 'eval_stale':
      return 'warning';
    case 'eval_unavailable':
      return 'failed';
    default:
      return 'pending';
  }
}

export function mlEvalStatusLabel(status?: MlEvalStatus): string {
  switch (status) {
    case 'healthy':
      return 'Healthy';
    case 'drift_detected':
      return 'Drift detected';
    case 'eval_stale':
      return 'Eval stale';
    case 'eval_unavailable':
      return 'Eval unavailable';
    default:
      return 'Unknown';
  }
}

export function formatMlEvalAge(
  generatedAt?: string,
  warningHours = 24
): { label: string; warning: boolean } {
  if (!generatedAt) {
    return { label: 'No eval timestamp', warning: true };
  }
  const parsed = new Date(generatedAt);
  if (Number.isNaN(parsed.getTime())) {
    return { label: generatedAt, warning: true };
  }
  const ageMs = Date.now() - parsed.getTime();
  const ageHours = ageMs / (1000 * 60 * 60);
  if (ageHours < 1) {
    return { label: 'Updated less than 1h ago', warning: false };
  }
  if (ageHours < 24) {
    return { label: `Updated ${Math.round(ageHours)}h ago`, warning: ageHours > warningHours };
  }
  const ageDays = Math.round(ageHours / 24);
  return {
    label: `Updated ${ageDays}d ago`,
    warning: ageHours > warningHours,
  };
}

export function formatShadowPrecision(value?: number): string {
  if (value == null || value <= 0) return '—';
  return `${(value * 100).toFixed(1)}% (proxy)`;
}
