// ops blacklist directory: paginated list + add/remove mutations; coalesced refresh while saving.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { addOpsBlacklistEntry, listOpsBlacklist, removeOpsBlacklistEntry } from '@/api/ops_api';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';
import { parseListLimit, parseListOffset } from '@/lib/list_query';

export function useOpsBlacklistPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [draftIp, setDraftIp] = useState('');
  const [draftReason, setDraftReason] = useState('');
  const [draftRemoveIp, setDraftRemoveIp] = useState('');

  const limit = parseListLimit(searchParams.get('limit'));
  const offset = parseListOffset(searchParams.get('offset'));

  const {
    data,
    error,
    fetching,
    revalidating: listRevalidating,
  } = useResource(
    (signal) => listOpsBlacklist({ limit, offset }, signal),
    [limit, offset, refreshToken]
  );

  const listBusy = fetching || saving;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      next.set('offset', String(Math.max(0, nextOffset)));
      replaceSearchParams(next);
    },
    [limit, replaceSearchParams, searchParams]
  );

  const onAdd = useCallback(async () => {
    if (saving) {
      return;
    }
    const ip = draftIp.trim();
    if (!ip) {
      return;
    }
    setSaving(true);
    setActionError(undefined);
    try {
      await addOpsBlacklistEntry({
        ip,
        reason: draftReason.trim() || undefined,
      });
      setDraftIp('');
      setDraftReason('');
      toast.success('Blacklist entry added');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setSaving(false);
    }
  }, [draftIp, draftReason, bumpRefreshCoalesced, saving]);

  const onRemove = useCallback(async () => {
    if (saving) {
      return;
    }
    const ip = draftRemoveIp.trim();
    if (!ip) {
      return;
    }
    if (!confirmDestructiveAction(`Remove ${ip} from the fraud blacklist?`)) {
      return;
    }
    setSaving(true);
    setActionError(undefined);
    try {
      await removeOpsBlacklistEntry({ ip });
      setDraftRemoveIp('');
      toast.success('Blacklist entry removed');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      const nextError = mutationError(err);
      setActionError(nextError);
      toast.error(nextError.message);
    } finally {
      setSaving(false);
    }
  }, [draftRemoveIp, bumpRefreshCoalesced, saving]);

  return {
    items: data?.items,
    total: data?.total ?? 0,
    limit,
    offset,
    draftIp,
    draftReason,
    draftRemoveIp,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    saving,
    error,
    actionError,
    hasSnapshot: data != null,
    onDraftIpChange: setDraftIp,
    onDraftReasonChange: setDraftReason,
    onDraftRemoveIpChange: setDraftRemoveIp,
    onAdd,
    onRemove,
    onPageChange,
  };
}
