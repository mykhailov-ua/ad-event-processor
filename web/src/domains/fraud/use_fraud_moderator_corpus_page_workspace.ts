import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import {
  importModeratorCorpus,
  listModeratorCorpus,
  previewModeratorCorpus,
  upsertModeratorCorpus,
} from '@/api/fraud_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { parseListLimit, parseListOffset } from '@/lib/list_query';
import { mutationError } from '@/lib/mutation_audit';

export function useFraudModeratorCorpusPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<Error | undefined>();
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState<Error | undefined>();
  const [previewCount, setPreviewCount] = useState<number | undefined>();
  const [previewError, setPreviewError] = useState<Error | undefined>();
  const [draftJa3, setDraftJa3] = useState('');
  const [draftJa4, setDraftJa4] = useState('');
  const [draftTcpSig, setDraftTcpSig] = useState('');
  const [draftWebgl, setDraftWebgl] = useState('');
  const [draftDesync, setDraftDesync] = useState('');
  const [draftNote, setDraftNote] = useState('');
  const [draftCsv, setDraftCsv] = useState('');

  const appliedLimit = parseListLimit(searchParams.get('limit'), 50);
  const appliedOffset = parseListOffset(searchParams.get('offset'));

  const { data, error, fetching, revalidating } = useResource(
    (signal) =>
      listModeratorCorpus({ limit: appliedLimit, offset: appliedOffset }, signal),
    [appliedLimit, appliedOffset, refreshToken]
  );

  const listBusy = fetching || saving || importing;
  const bumpList = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onSaveTuple = useCallback(async () => {
    const ja3 = draftJa3.trim();
    if (!ja3) {
      setSaveError(new Error('JA3 is required'));
      return;
    }
    setSaving(true);
    setSaveError(undefined);
    try {
      await upsertModeratorCorpus({
        ja3,
        ja4: draftJa4.trim() || undefined,
        tcp_sig: draftTcpSig.trim() || undefined,
        webgl_renderer: draftWebgl.trim() || undefined,
        layer_desync_count: draftDesync.trim() ? Number(draftDesync) : undefined,
        note: draftNote.trim() || undefined,
      });
      toast.success('Corpus tuple saved');
      bumpList();
    } catch (err) {
      setSaveError(mutationError(err));
    } finally {
      setSaving(false);
    }
  }, [bumpList, draftDesync, draftJa3, draftJa4, draftNote, draftTcpSig, draftWebgl]);

  const onImportCsv = useCallback(async () => {
    const csv = draftCsv.trim();
    if (!csv) {
      setImportError(new Error('CSV body is required'));
      return;
    }
    setImporting(true);
    setImportError(undefined);
    try {
      const result = await importModeratorCorpus({ csv });
      toast.success(`Imported ${result.upserted} tuple(s)`);
      bumpList();
    } catch (err) {
      setImportError(mutationError(err));
    } finally {
      setImporting(false);
    }
  }, [bumpList, draftCsv]);

  const onPreview = useCallback(async () => {
    const ja3 = draftJa3.trim();
    if (!ja3) {
      setPreviewError(new Error('JA3 is required for preview'));
      return;
    }
    setPreviewError(undefined);
    try {
      const result = await previewModeratorCorpus(ja3);
      setPreviewCount(result.match_count_7d);
    } catch (err) {
      setPreviewCount(undefined);
      setPreviewError(mutationError(err));
    }
  }, [draftJa3]);

  const onLimitChange = useCallback(
    (limit: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      next.set('offset', '0');
      replaceSearchParams(next);
    },
    [replaceSearchParams, searchParams]
  );

  const onOffsetChange = useCallback(
    (offset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('offset', String(Math.max(0, offset)));
      replaceSearchParams(next);
    },
    [replaceSearchParams, searchParams]
  );

  return {
    items: data?.items,
    total: data?.total ?? 0,
    limit: appliedLimit,
    offset: appliedOffset,
    lastRefresh: data?.last_refresh,
    fetching: fetching || listQueryPending,
    revalidating,
    error,
    saving,
    saveError,
    importing,
    importError,
    previewCount,
    previewError,
    draftJa3,
    draftJa4,
    draftTcpSig,
    draftWebgl,
    draftDesync,
    draftNote,
    draftCsv,
    setDraftJa3,
    setDraftJa4,
    setDraftTcpSig,
    setDraftWebgl,
    setDraftDesync,
    setDraftNote,
    setDraftCsv,
    onSaveTuple,
    onImportCsv,
    onPreview,
    onLimitChange,
    onOffsetChange,
  };
}
