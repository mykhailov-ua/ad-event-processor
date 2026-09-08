// audit log directory: paginated list + CSV export with optional PII redaction flag.
import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { exportAuditCsv, listAudit } from '@/api/audit_api';
import type { AuditListQuery } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

function buildListQuery(params: URLSearchParams): AuditListQuery {
  return {
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
  };
}

export function useAuditPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const query = useMemo(() => buildListQuery(searchParams), [searchParams]);

  const [draftRedactPii, setDraftRedactPii] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<Error | undefined>();
  const [exportTruncated, setExportTruncated] = useState(false);
  const [exportNextCursor, setExportNextCursor] = useState<string | undefined>();

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource((signal) => listAudit(query, signal), [query.limit, query.offset]);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(query.limit ?? DEFAULT_LIST_LIMIT));
      next.set('offset', String(Math.max(0, nextOffset)));
      replaceSearchParams(next);
    },
    [query.limit, replaceSearchParams, searchParams]
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
      toast.success('Audit CSV exported');
    } catch (err: unknown) {
      const nextError = err instanceof Error ? err : new Error(String(err));
      setExportError(nextError);
      toast.error(nextError.message);
    } finally {
      setExporting(false);
    }
  }, [draftRedactPii]);

  return {
    items: data?.items,
    total: data?.total ?? 0,
    limit: query.limit ?? DEFAULT_LIST_LIMIT,
    offset: query.offset ?? 0,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error,
    hasSnapshot: data != null,
    draftRedactPii,
    exporting,
    exportError,
    exportTruncated,
    exportNextCursor,
    onDraftRedactPiiChange: setDraftRedactPii,
    onExportCsv,
    onPageChange,
  };
}
