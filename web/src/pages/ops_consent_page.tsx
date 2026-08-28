import { useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { buildConsentProofsUrl, type ConsentProofList } from '../helpers/ops_api.js';
import { useResource } from '../helpers/use_resource.js';
import { OpsConsentPanel } from '../ui/ops/ops_consent_panel.js';

const PAGE_LIMIT = 50;

export function OpsConsentPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const userIdFilter = searchParams.get('user_id') ?? '';
  const cursor = searchParams.get('cursor') ?? '';
  const [draftUserId, setDraftUserId] = useState(userIdFilter);

  const url = useMemo(
    () =>
      buildConsentProofsUrl({
        limit: PAGE_LIMIT,
        cursor: cursor || undefined,
        user_id: userIdFilter || undefined,
      }),
    [cursor, userIdFilter]
  );

  const { data, loading, error } = useResource<ConsentProofList>(url);

  const patchParams = useCallback(
    (patch: Record<string, string | null>) => {
      const next = new URLSearchParams(searchParams);
      for (const [key, value] of Object.entries(patch)) {
        if (value === null || value === '') next.delete(key);
        else next.set(key, value);
      }
      setSearchParams(next, { replace: true });
    },
    [searchParams, setSearchParams]
  );

  return (
    <OpsConsentPanel
      data={data}
      loading={loading}
      error={error}
      userIdFilter={draftUserId}
      onUserIdFilterChange={setDraftUserId}
      onApplyFilter={() =>
        patchParams({ user_id: draftUserId.trim() || null, cursor: null })
      }
      cursor={cursor}
      nextCursor={data?.next_cursor ?? null}
      onPrevCursor={() => patchParams({ cursor: null })}
      onNextCursor={() => {
        if (!data?.next_cursor) return;
        patchParams({ cursor: data.next_cursor });
      }}
    />
  );
}
