import type { ExportHubKind } from '@/domains/exports/export_hub_catalog';

export type ExportHubRecentJob = {
  jobId: string;
  kind: ExportHubKind;
  label: string;
  customerId?: string;
  rowLimit?: number;
  status: string;
  bytes?: number;
  error?: string;
  createdAt: string;
};

const STORAGE_KEY = 'aed.export_hub.recent.v1';
export const EXPORT_HUB_RECENT_MAX = 5;

function readRecentRaw(): ExportHubRecentJob[] {
  if (typeof sessionStorage === 'undefined') {
    return [];
  }
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter(
      (entry): entry is ExportHubRecentJob =>
        entry != null &&
        typeof entry === 'object' &&
        typeof (entry as ExportHubRecentJob).jobId === 'string' &&
        typeof (entry as ExportHubRecentJob).kind === 'string' &&
        typeof (entry as ExportHubRecentJob).label === 'string' &&
        typeof (entry as ExportHubRecentJob).status === 'string' &&
        typeof (entry as ExportHubRecentJob).createdAt === 'string'
    );
  } catch {
    return [];
  }
}

function writeRecentRaw(entries: ExportHubRecentJob[]): void {
  if (typeof sessionStorage === 'undefined') {
    return;
  }
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(entries.slice(0, EXPORT_HUB_RECENT_MAX)));
}

export function listExportHubRecentJobs(): ExportHubRecentJob[] {
  return readRecentRaw();
}

export function upsertExportHubRecentJob(entry: ExportHubRecentJob): ExportHubRecentJob[] {
  const without = readRecentRaw().filter((row) => row.jobId !== entry.jobId);
  const next = [entry, ...without].slice(0, EXPORT_HUB_RECENT_MAX);
  writeRecentRaw(next);
  return next;
}

export function patchExportHubRecentJob(
  jobId: string,
  patch: Partial<Omit<ExportHubRecentJob, 'jobId'>>
): ExportHubRecentJob[] {
  const next = readRecentRaw().map((row) =>
    row.jobId === jobId ? { ...row, ...patch, jobId: row.jobId } : row
  );
  writeRecentRaw(next);
  return next;
}
