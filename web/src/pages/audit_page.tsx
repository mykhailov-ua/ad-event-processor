import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { buildAuditExportUrl, listAudit, type AuditLog } from '../helpers/audit_api.js';
import { downloadBlob, fetchBlob } from '../helpers/api_blob.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { to } from '../lib/to.js';
import { AuditDirectory } from '../ui/audit/audit_directory.js';

const DEFAULT_LIMIT = 50;

function parseLimit(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value <= 0) return DEFAULT_LIMIT;
  return Math.min(value, 200);
}

function parseOffset(raw: string | null): number {
  const value = Number.parseInt(raw ?? '', 10);
  if (!Number.isFinite(value) || value < 0) return 0;
  return value;
}

function parseRedactPii(raw: string | null): boolean {
  if (raw === 'false') return false;
  return true;
}

export function AuditPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const limit = parseLimit(searchParams.get('limit'));
  const offset = parseOffset(searchParams.get('offset'));
  const redactPii = parseRedactPii(searchParams.get('redact_pii'));

  const [items, setItems] = useState<AuditLog[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [exporting, setExporting] = useState(false);
  const [exportTruncated, setExportTruncated] = useState(false);
  const [exportNextCursor, setExportNextCursor] = useState<string | null>(null);

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') {
          next.delete(key);
        } else {
          next.set(key, value);
        }
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  useEffect(() => {
    const ctrl = new AbortController();
    let cancelled = false;
    setLoading(true);
    setError(null);

    void (async () => {
      const [result, err] = await to(
        listAudit({ limit, offset, redact_pii: redactPii }, ctrl.signal)
      );
      if (cancelled) return;
      if (err) {
        if (err.name === 'AbortError') return;
        setError(err);
        setLoading(false);
        return;
      }
      setItems(result?.items ?? []);
      setTotal(result?.total ?? 0);
      setLoading(false);
    })();

    return () => {
      cancelled = true;
      ctrl.abort();
    };
  }, [limit, offset, redactPii]);

  const onRedactPiiChange = useCallback(
    (value: boolean) => {
      patchParams({
        redact_pii: value ? 'true' : 'false',
        offset: '0',
      });
    },
    [patchParams]
  );

  const onOffsetChange = useCallback(
    (nextOffset: number) => {
      patchParams({ offset: String(nextOffset) });
    },
    [patchParams]
  );

  const runExport = useCallback(
    async (cursor?: string) => {
      setExporting(true);
      try {
        const path = buildAuditExportUrl({ redact_pii: redactPii, cursor });
        const result = await fetchBlob(path);
        downloadBlob(result.blob, 'audit-export.csv');
        setExportTruncated(result.truncated);
        setExportNextCursor(result.nextCursor);
        pushToastMessage({ title: 'Export ready', message: 'audit-export.csv downloaded' });
      } catch (err) {
        setExportTruncated(false);
        setExportNextCursor(null);
        pushToastMessage({
          title: 'Export failed',
          message: err instanceof Error ? err.message : 'Export failed',
        });
      } finally {
        setExporting(false);
      }
    },
    [redactPii]
  );

  const onExport = useCallback(() => {
    void runExport();
  }, [runExport]);

  const onContinueExport = useCallback(() => {
    if (!exportNextCursor) return;
    void runExport(exportNextCursor);
  }, [exportNextCursor, runExport]);

  const onDismissExportBanner = useCallback(() => {
    setExportTruncated(false);
    setExportNextCursor(null);
  }, []);

  return (
    <AuditDirectory
      items={items}
      total={total}
      limit={limit}
      offset={offset}
      redactPii={redactPii}
      loading={loading}
      error={error}
      exporting={exporting}
      exportTruncated={exportTruncated}
      exportNextCursor={exportNextCursor}
      onRedactPiiChange={onRedactPiiChange}
      onOffsetChange={onOffsetChange}
      onExport={onExport}
      onContinueExport={onContinueExport}
      onDismissExportBanner={onDismissExportBanner}
    />
  );
}
