import { api } from './api_client.js';

export type MLModelVersion = {
  id: string;
  artifact_hash: string;
  status: string;
  created_at: string;
  artifact_metadata?: Record<string, unknown>;
};

export type MLModelRedis = {
  version_id?: string;
  hash?: string;
  applied_at?: string;
  shards_reporting: number;
  shards_consistent: boolean;
};

export type MLShardSync = {
  shard_id: number;
  model_version: string;
  phase: string;
  started_at: string;
};

export type MLFeatureImportance = {
  name: string;
  value: number;
};

export type MLModelStatus = {
  active_version?: MLModelVersion | null;
  syncing_version?: MLModelVersion | null;
  redis: MLModelRedis;
  shard_sync: MLShardSync[];
  drift?: unknown;
  drift_detected?: boolean;
  precision?: number;
  recall?: number;
  importance?: MLFeatureImportance[];
};

export type MLEvalMetricsBlock = {
  status: string;
  label_method?: string;
  label_definition?: string;
  labeled_rows: number;
  matched_rows?: number;
  confidence?: string;
  precision?: number;
  recall?: number;
  f1?: number;
  false_positive_rate?: number;
  tp?: number;
  fp?: number;
  fn?: number;
  tn?: number;
};

export type MLEvalReport = {
  status: string;
  generated_at?: string;
  hours?: number;
  threshold?: number;
  proxy_metrics: MLEvalMetricsBlock;
  audited_metrics: MLEvalMetricsBlock;
  drift?: unknown;
  drift_detected?: boolean;
};

export type OpsMLManualLabel = {
  ip_hash: string;
  label: number;
  reason?: string;
  source?: string;
  created_at?: string;
};

const OPS_ML_POLL_MS = 30_000;

export function opsMlModelPollMs(): number {
  return OPS_ML_POLL_MS;
}

export async function fetchOpsMLModelStatus(): Promise<MLModelStatus | null> {
  const res = await api<MLModelStatus>('/api/v1/ops/ml-model');
  return res.data ?? null;
}

export async function fetchOpsMLEvalReport(): Promise<MLEvalReport | null> {
  const res = await api<MLEvalReport>('/api/v1/ops/ml-model/eval');
  return res.data ?? null;
}

export async function fetchOpsMLManualLabels(): Promise<OpsMLManualLabel[]> {
  const res = await api<OpsMLManualLabel[]>('/api/v1/ops/ml-model/labels');
  return Array.isArray(res.data) ? res.data : [];
}

export function truncateArtifactHash(hash?: string, head = 8, tail = 8): string {
  if (!hash) return '—';
  if (hash.length <= head + tail + 1) return hash;
  return `${hash.slice(0, head)}…${hash.slice(-tail)}`;
}
