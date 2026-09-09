// audit log directory: paginated list + CSV export with optional PII redaction flag.
import { useCallback, useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { exportAuditCsv, listAudit } from '@/api/audit_api';
import type { AuditListQuery } from '@/api/types';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { DEFAULT_LIST_LIMIT, parseListLimit, parseListOffset } from '@/lib/list_query';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';
import { toError } from '@/lib/admin_error';

export type AuditAuthSourceFilter = '' | 'session' | 'api_key';

function parseAuthSource(raw: string | null): AuditAuthSourceFilter {
  if (raw === 'session' || raw === 'api_key') {
    return raw;
  }
  return '';
}

function buildListQuery(params: URLSearchParams): AuditListQuery {
  const adminId = params.get('admin_id')?.trim();
  const targetId = (params.get('target_id') ?? params.get('campaign_id'))?.trim();
  const action = params.get('action')?.trim();
  const authSource = parseAuthSource(params.get('auth_source'));
  const apiKeyId = params.get('api_key_id')?.trim();

  return {
    limit: parseListLimit(params.get('limit')),
    offset: parseListOffset(params.get('offset')),
    admin_id: adminId || undefined,
    target_id: targetId || undefined,
    action: action || undefined,
    auth_source: authSource || undefined,
    api_key_id: apiKeyId || undefined,
  };
}

export function useAuditPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const query = useMemo(() => buildListQuery(searchParams), [searchParams]);

  const appliedAdminId = query.admin_id ?? '';
  const appliedTargetId = query.target_id ?? '';
  const appliedAction = query.action ?? '';
  const appliedAuthSource = parseAuthSource(query.auth_source ?? null);
  const appliedApiKeyId = query.api_key_id ?? '';

  const [draftAdminId, setDraftAdminId] = useState(appliedAdminId);
  const [draftTargetId, setDraftTargetId] = useState(appliedTargetId);
  const [draftAction, setDraftAction] = useState(appliedAction);
  const [draftAuthSource, setDraftAuthSource] = useState<AuditAuthSourceFilter>(appliedAuthSource);
  const [draftApiKeyId, setDraftApiKeyId] = useState(appliedApiKeyId);
  const [draftRedactPii, setDraftRedactPii] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [exportError, setExportError] = useState<Error | undefined>();
  const [exportTruncated, setExportTruncated] = useState(false);
  const [exportNextCursor, setExportNextCursor] = useState<string | undefined>();

  useEffect(() => {
    setDraftAdminId(appliedAdminId);
    setDraftTargetId(appliedTargetId);
    setDraftAction(appliedAction);
    setDraftAuthSource(appliedAuthSource);
    setDraftApiKeyId(appliedApiKeyId);
  }, [appliedAction, appliedAdminId, appliedApiKeyId, appliedAuthSource, appliedTargetId]);

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => listAudit(query, signal),
    [
      query.limit,
      query.offset,
      query.admin_id,
      query.target_id,
      query.action,
      query.auth_source,
      query.api_key_id,
    ]
  );

  const updateQuery = useCallback(
    (patch: Partial<AuditListQuery>) => {
      const next = new URLSearchParams(searchParams);
      const merged = { ...query, ...patch };

      next.set('limit', String(merged.limit ?? DEFAULT_LIST_LIMIT));
      next.set('offset', String(merged.offset ?? 0));

      if (merged.admin_id) {
        next.set('admin_id', merged.admin_id);
      } else {
        next.delete('admin_id');
      }

      if (merged.target_id) {
        next.set('target_id', merged.target_id);
      } else {
        next.delete('target_id');
        next.delete('campaign_id');
      }

      if (merged.action) {
        next.set('action', merged.action);
      } else {
        next.delete('action');
      }

      if (merged.auth_source) {
        next.set('auth_source', merged.auth_source);
      } else {
        next.delete('auth_source');
      }

      if (merged.api_key_id) {
        next.set('api_key_id', merged.api_key_id);
      } else {
        next.delete('api_key_id');
      }

      replaceSearchParams(next);
    },
    [query, replaceSearchParams, searchParams]
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      updateQuery({ offset: Math.max(0, nextOffset) });
    },
    [updateQuery]
  );

  const onApplyFilters = useCallback(() => {
    updateQuery({
      admin_id: draftAdminId.trim() || undefined,
      target_id: draftTargetId.trim() || undefined,
      action: draftAction.trim() || undefined,
      auth_source: draftAuthSource || undefined,
      api_key_id: draftApiKeyId.trim() || undefined,
      offset: 0,
    });
  }, [
    draftAction,
    draftAdminId,
    draftApiKeyId,
    draftAuthSource,
    draftTargetId,
    updateQuery,
  ]);

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
      setExportError(toError(err));
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
    draftAdminId,
    draftTargetId,
    draftAction,
    draftAuthSource,
    draftApiKeyId,
    draftRedactPii,
    exporting,
    exportError,
    exportTruncated,
    exportNextCursor,
    onDraftAdminIdChange: setDraftAdminId,
    onDraftTargetIdChange: setDraftTargetId,
    onDraftActionChange: setDraftAction,
    onDraftAuthSourceChange: setDraftAuthSource,
    onDraftApiKeyIdChange: setDraftApiKeyId,
    onDraftRedactPiiChange: setDraftRedactPii,
    onApplyFilters,
    onExportCsv,
    onPageChange,
  };
}
