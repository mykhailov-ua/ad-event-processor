// RTB floors apply tool: POST placement id list; dry_run default true.
import { useCallback, useState } from 'react';

import { applyRtbFloors } from '@/api/rtb_api';
import type { RtbFloorsApplyResult } from '@/api/types';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';

export function useRtbFloorsPageWorkspace() {
  const [draftPlacementIds, setDraftPlacementIds] = useState('');
  const [dryRun, setDryRun] = useState(true);
  const [result, setResult] = useState<RtbFloorsApplyResult | undefined>();
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<Error | undefined>();
  const [licenseGated, setLicenseGated] = useState(false);

  const onApply = useCallback(async () => {
    setApplying(true);
    setError(undefined);
    setLicenseGated(false);
    try {
      const placementIds = draftPlacementIds
        .split('\n')
        .map((line) => line.trim())
        .filter(Boolean);
      const next = await applyRtbFloors(
        placementIds.length > 0 ? { placement_ids: placementIds } : {},
        dryRun
      );
      setResult(next);
    } catch (err: unknown) {
      const wrapped = err instanceof Error ? err : new Error(String(err));
      setError(wrapped);
      setLicenseGated(rtbLicenseGated(wrapped));
    } finally {
      setApplying(false);
    }
  }, [draftPlacementIds, dryRun]);

  const onApplyCoalesced = useCoalescedCallback(
    () => {
      void onApply();
    },
    { inFlightGuard: true, inFlight: applying }
  );

  return {
    result,
    draftPlacementIds,
    dryRun,
    applying,
    error: licenseGated ? undefined : error,
    licenseGated,
    onDraftPlacementIdsChange: setDraftPlacementIds,
    onDryRunChange: setDryRun,
    onApply: onApplyCoalesced,
  };
}
