import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';

import { getRtbReconcileExport, getRtbShadowDiff } from '@/api/rtb_api';
import { RtbShadowTools } from '@/domains/rtb/rtb_shadow_tools';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useResource } from '@/api/use_resource';

export function RtbShadowPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const appliedWindow = searchParams.get('window') ?? '1h';
  const appliedRequestId = searchParams.get('request_id') ?? '';

  const [draftWindow, setDraftWindow] = useState(appliedWindow);
  const [draftRequestId, setDraftRequestId] = useState(appliedRequestId);

  useEffect(() => {
    setDraftWindow(appliedWindow);
    setDraftRequestId(appliedRequestId);
  }, [appliedRequestId, appliedWindow]);

  const { data, error, fetching } = useResource(
    async (signal) => {
      const [shadow, reconcile] = await Promise.all([
        getRtbShadowDiff(appliedWindow, signal),
        getRtbReconcileExport(
          {
            window: appliedWindow,
            request_id: appliedRequestId || undefined,
          },
          signal,
        ),
      ]);
      return { shadow, reconcile };
    },
    [appliedRequestId, appliedWindow],
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
    setSearchParams(next, { replace: true });
  }, [draftRequestId, draftWindow, searchParams, setSearchParams]);

  return (
    <RtbShadowTools
      shadow={data?.shadow}
      reconcile={data?.reconcile}
      draftWindow={draftWindow}
      draftRequestId={draftRequestId}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={data != null || licenseGated}
      licenseGated={licenseGated}
      onDraftWindowChange={setDraftWindow}
      onDraftRequestIdChange={setDraftRequestId}
      onApply={onApply}
    />
  );
}
