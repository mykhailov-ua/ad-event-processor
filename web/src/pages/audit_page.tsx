import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { exportAuditCsv, listAudit } from '@/api/audit_api';
import type { AuditListQuery } from '@/api/types';
import { AuditDirectory } from '@/domains/audit/audit_directory';
import { useResource } from '@/api/use_resource';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';

function buildListQuery(params: URLSearchParams): AuditListQuery {
  return {
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
  };
}

function triggerBlobDownload(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
  URL.revokeObjectURL(url);
}

export function AuditPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const query = useMemo(() => buildListQuery(searchParams), [searchParams]);

  const [draftRedactPii, setDraftRedactPii] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<Error | undefined>();
  const [exportTruncated, setExportTruncated] = useState(false);
  const [exportNextCursor, setExportNextCursor] = useState<string | undefined>();

  const { data, error, fetching } = useResource(
    (signal) => listAudit(query, signal),
    [query.limit, query.offset],
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(query.limit ?? DEFAULT_LIST_LIMIT));
      next.set('offset', String(Math.max(0, nextOffset)));
      setSearchParams(next, { replace: true });
    },
    [query.limit, searchParams, setSearchParams],
  );

  const onExportCsv = useCallback(async () => {
    setExporting(true);
    setExportError(undefined);
    setExportTruncated(false);
    setExportNextCursor(undefined);
    try {
      const result = await exportAuditCsv({
        format: 'csv',
        redact_pii: draftRedactPii || undefined,
      });
      triggerBlobDownload(result.blob, 'audit-export.csv');
      setExportTruncated(result.truncated);
      setExportNextCursor(result.nextCursor);
    } catch (err) {
      setExportError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setExporting(false);
    }
  }, [draftRedactPii]);

  return (
    <AuditDirectory
      items={data?.items ?? []}
      total={data?.total ?? 0}
      limit={query.limit ?? DEFAULT_LIST_LIMIT}
      offset={query.offset ?? 0}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      draftRedactPii={draftRedactPii}
      exporting={exporting}
      exportError={exportError}
      exportTruncated={exportTruncated}
      exportNextCursor={exportNextCursor}
      onDraftRedactPiiChange={setDraftRedactPii}
      onExportCsv={onExportCsv}
      onPageChange={onPageChange}
    />
  );
}
