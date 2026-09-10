import {
  adminMetricMutedZeroClass,
  adminMetricNegativeClass,
  adminMetricPositiveClass,
  adminOpsWarnClass,
} from '@/lib/admin_metric_tone';
import type { AdminStatusTone } from '@/lib/admin_kit';

export type ExportHubJobPhase = 'pending' | 'completed' | 'failed' | 'cancelled' | 'unknown';

export function normalizeExportJobStatus(status: string | undefined): string {
  return (status ?? '').trim().toLowerCase();
}

export function exportJobStatusDisplayLabel(status: string | undefined): string {
  const normalized = normalizeExportJobStatus(status);
  if (!normalized) {
    return 'No status yet';
  }

  const phase = exportJobPhase(status);
  if (phase === 'pending') {
    return 'Running';
  }
  if (phase === 'completed') {
    return 'Completed';
  }
  if (phase === 'failed') {
    return 'Failed';
  }
  if (phase === 'cancelled') {
    return 'Cancelled';
  }

  return normalized;
}

export function exportJobAdminStatusTone(status: string | undefined): AdminStatusTone {
  const phase = exportJobPhase(status);
  if (phase === 'completed') {
    return 'active';
  }
  if (phase === 'failed') {
    return 'error';
  }
  if (phase === 'pending') {
    return 'paused';
  }
  if (phase === 'cancelled') {
    return 'archived';
  }
  return 'muted';
}

export function exportJobStatusTextClass(status: string | undefined): string {
  const phase = exportJobPhase(status);
  if (phase === 'completed') {
    return adminMetricPositiveClass;
  }
  if (phase === 'failed') {
    return adminMetricNegativeClass;
  }
  if (phase === 'pending') {
    return adminOpsWarnClass;
  }
  return adminMetricMutedZeroClass;
}

export function exportJobPhase(status: string | undefined): ExportHubJobPhase {
  const normalized = normalizeExportJobStatus(status);
  if (
    normalized === 'pending' ||
    normalized === 'running' ||
    normalized === 'queued' ||
    normalized === 'processing'
  ) {
    return 'pending';
  }
  if (normalized === 'completed' || normalized === 'done') {
    return 'completed';
  }
  if (normalized === 'failed' || normalized === 'error') {
    return 'failed';
  }
  if (normalized === 'cancelled' || normalized === 'canceled') {
    return 'cancelled';
  }
  return 'unknown';
}

export function exportJobCanDownload(status: string | undefined): boolean {
  return exportJobPhase(status) === 'completed';
}

export function exportJobCanCancel(status: string | undefined): boolean {
  return exportJobPhase(status) === 'pending';
}

export function formatExportJobBytes(bytes: number | undefined): string | undefined {
  if (bytes == null || !Number.isFinite(bytes) || bytes <= 0) {
    return undefined;
  }
  const units = ['B', 'KB', 'MB', 'GB'] as const;
  let value = bytes;
  let unitIndex = 0;
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }
  const digits = unitIndex === 0 ? 0 : value >= 100 ? 1 : 1;
  return `${value.toFixed(digits)} ${units[unitIndex]}`;
}

export function formatExportJobRowSummary(
  rowLimit: number | undefined,
  bytes: number | undefined
): string | undefined {
  const sizeLabel = formatExportJobBytes(bytes);
  const rowLabel =
    rowLimit != null && Number.isFinite(rowLimit) && rowLimit > 0
      ? `${rowLimit.toLocaleString()} row limit`
      : undefined;
  if (sizeLabel && rowLabel) {
    return `${sizeLabel} (${rowLabel})`;
  }
  return sizeLabel ?? rowLabel;
}

export const EXPORT_JOB_INLINE_TEXT_MAX = 120;

export type ExportJobInlineText = {
  display: string;
  full: string;
  truncated: boolean;
};

export function truncateExportJobInlineText(
  text: string,
  maxLength = EXPORT_JOB_INLINE_TEXT_MAX
): ExportJobInlineText {
  const trimmed = text.trim();
  if (trimmed.length <= maxLength) {
    return { display: trimmed, full: trimmed, truncated: false };
  }
  return {
    display: `${trimmed.slice(0, maxLength - 3)}...`,
    full: trimmed,
    truncated: true,
  };
}

export function formatExportJobRecentSummary(
  status: string | undefined,
  rowLimit: number | undefined,
  bytes: number | undefined
): string | undefined {
  if (exportJobPhase(status) === 'failed') {
    return formatExportJobBytes(bytes);
  }
  return formatExportJobRowSummary(rowLimit, bytes);
}

export function shortExportJobId(jobId: string): string {
  const trimmed = jobId.trim();
  if (trimmed.length <= 8) {
    return trimmed;
  }
  return `${trimmed.slice(0, 8)}...`;
}

export function formatExportJobElapsed(startedAtMs: number, nowMs: number): string {
  const elapsedSec = Math.max(0, Math.floor((nowMs - startedAtMs) / 1000));
  const minutes = Math.floor(elapsedSec / 60);
  const seconds = elapsedSec % 60;
  if (minutes > 0) {
    return `${minutes}m ${String(seconds).padStart(2, '0')}s`;
  }
  return `${seconds}s`;
}
