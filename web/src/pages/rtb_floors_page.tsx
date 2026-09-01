import { useCallback, useState } from 'react';

import { applyRtbFloors } from '@/api/rtb_api';
import type { RtbFloorsApplyResult } from '@/api/types';
import { RtbFloorsApplyPanel } from '@/domains/rtb/rtb_floors_apply';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';

export function RtbFloorsPage() {
  const [draftPlacementIds, setDraftPlacementIds] = useState('');
  const [dryRun, setDryRun] = useState(true);
  const [result, setResult] = useState<RtbFloorsApplyResult | undefined>(undefined);
  const [applying, setApplying] = useState(false);
  const [error, setError] = useState<Error | undefined>(undefined);
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
        dryRun,
      );
      setResult(next);
    } catch (err) {
      const wrapped = err instanceof Error ? err : new Error(String(err));
      setError(wrapped);
      setLicenseGated(rtbLicenseGated(wrapped));
    } finally {
      setApplying(false);
    }
  }, [draftPlacementIds, dryRun]);

  return (
    <RtbFloorsApplyPanel
      result={result}
      draftPlacementIds={draftPlacementIds}
      dryRun={dryRun}
      applying={applying}
      error={licenseGated ? undefined : error}
      licenseGated={licenseGated}
      onDraftPlacementIdsChange={setDraftPlacementIds}
      onDryRunChange={setDryRun}
      onApply={() => {
        void onApply();
      }}
    />
  );
}
