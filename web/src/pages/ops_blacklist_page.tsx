import { useCallback, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import {
  addOpsBlacklistEntry,
  listOpsBlacklist,
  removeOpsBlacklistEntry,
} from '@/api/ops_api';
import { OpsBlacklist } from '@/domains/ops/ops_blacklist';
import { useResource } from '@/api/use_resource';
import { parseListLimit, parseListOffset } from '@/lib/list_query';

export function OpsBlacklistPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const [refreshToken, setRefreshToken] = useState(0);
  const [saving, setSaving] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [draftIp, setDraftIp] = useState('');
  const [draftReason, setDraftReason] = useState('');
  const [draftRemoveIp, setDraftRemoveIp] = useState('');

  const limit = parseListLimit(searchParams.get('limit'));
  const offset = parseListOffset(searchParams.get('offset'));

  const { data, error, fetching } = useResource(
    (signal) => listOpsBlacklist({ limit, offset }, signal),
    [limit, offset, refreshToken],
  );

  const onPageChange = useCallback(
    (nextOffset: number) => {
      const next = new URLSearchParams(searchParams);
      next.set('limit', String(limit));
      next.set('offset', String(Math.max(0, nextOffset)));
      setSearchParams(next, { replace: true });
    },
    [limit, searchParams, setSearchParams],
  );

  const onAdd = useCallback(async () => {
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
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [draftIp, draftReason]);

  const onRemove = useCallback(async () => {
    const ip = draftRemoveIp.trim();
    if (!ip) {
      return;
    }
    setSaving(true);
    setActionError(undefined);
    try {
      await removeOpsBlacklistEntry({ ip });
      setDraftRemoveIp('');
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setActionError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setSaving(false);
    }
  }, [draftRemoveIp]);

  return (
    <OpsBlacklist
      items={data?.items ?? []}
      total={data?.total ?? 0}
      limit={limit}
      offset={offset}
      draftIp={draftIp}
      draftReason={draftReason}
      draftRemoveIp={draftRemoveIp}
      fetching={fetching}
      saving={saving}
      error={error}
      actionError={actionError}
      hasSnapshot={data != null}
      onDraftIpChange={setDraftIp}
      onDraftReasonChange={setDraftReason}
      onDraftRemoveIpChange={setDraftRemoveIp}
      onAdd={onAdd}
      onRemove={onRemove}
      onPageChange={onPageChange}
    />
  );
}
