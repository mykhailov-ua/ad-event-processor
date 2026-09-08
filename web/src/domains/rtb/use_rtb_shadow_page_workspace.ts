// L3 RTB shadow reconcile: parallel GET shadow diff + reconcile export; window/request_id in URL.
import { useCallback, useEffect, useState } from 'react';

import { getRtbReconcileExport, getRtbShadowDiff } from '@/api/rtb_api';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useResource } from '@/api/use_resource';
import { useTransitionSearchParams } from '@/hooks/use_transition_search_params';

export function useRtbShadowPageWorkspace() {
  const [searchParams, { isPending: listQueryPending, replaceSearchParams }] =
    useTransitionSearchParams();
  const appliedWindow = searchParams.get('window') ?? '1h';
  const appliedRequestId = searchParams.get('request_id') ?? '';

  const [draftWindow, setDraftWindow] = useState(appliedWindow);
  const [draftRequestId, setDraftRequestId] = useState(appliedRequestId);

  useEffect(() => {
    setDraftWindow(appliedWindow);
    setDraftRequestId(appliedRequestId);
  }, [appliedRequestId, appliedWindow]);

  const { data, error, fetching, revalidating: listRevalidating } = useResource(
    async (signal) => {
      const [shadow, reconcile] = await Promise.all([
        getRtbShadowDiff(appliedWindow, signal),
        getRtbReconcileExport(
          {
            window: appliedWindow,
            request_id: appliedRequestId || undefined,
          },
          signal
        ),
      ]);
      return { shadow, reconcile };
    },
    [appliedRequestId, appliedWindow]
  );

  const licenseGated = rtbLicenseGated(error);

  const onApply = useCallback(() => {
    const next = new URLSearchParams(searchParams);
    const window = draftWindow.trim() || '1h';
    next.set('window', window);
    const requestId = draftRequestId.trim();
    if (requestId) {
      next.set('request_id', requestId);
    } else {
      next.delete('request_id');
    }
    replaceSearchParams(next);
  }, [draftRequestId, draftWindow, replaceSearchParams, searchParams]);

  return {
    shadow: data?.shadow,
    reconcile: data?.reconcile,
    draftWindow,
    draftRequestId,
    fetching,
    listRevalidating: listRevalidating || listQueryPending,
    error: licenseGated ? undefined : error,
    hasSnapshot: data != null || licenseGated,
    licenseGated,
    onDraftWindowChange: setDraftWindow,
    onDraftRequestIdChange: setDraftRequestId,
    onApply,
  };
}
