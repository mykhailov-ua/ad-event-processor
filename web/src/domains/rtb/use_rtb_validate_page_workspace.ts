// L3 OpenRTB bid request validator: client JSON.parse then POST validate; license gate on 403.
import { useCallback, useState } from 'react';

import { validateRtbBidRequest } from '@/api/rtb_api';
import type { OpenRtbValidationResult } from '@/api/types';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';

export const RTB_VALIDATE_DEFAULT_JSON = '{\n  "id": "test-request"\n}';

export function useRtbValidatePageWorkspace() {
  const [draftJson, setDraftJson] = useState(RTB_VALIDATE_DEFAULT_JSON);
  const [result, setResult] = useState<OpenRtbValidationResult | undefined>();
  const [validating, setValidating] = useState(false);
  const [error, setError] = useState<Error | undefined>();
  const [licenseGated, setLicenseGated] = useState(false);

  const onValidate = useCallback(async () => {
    setValidating(true);
    setError(undefined);
    setLicenseGated(false);
    try {
      const parsed = JSON.parse(draftJson) as Record<string, unknown>;
      const next = await validateRtbBidRequest(parsed);
      setResult(next);
    } catch (err: unknown) {
      const wrapped = err instanceof Error ? err : new Error(String(err));
      setError(wrapped);
      setLicenseGated(rtbLicenseGated(wrapped));
    } finally {
      setValidating(false);
    }
  }, [draftJson]);

  const onValidateCoalesced = useCoalescedCallback(
    () => {
      void onValidate();
    },
    { inFlightGuard: true, inFlight: validating }
  );

  return {
    draftJson,
    result,
    validating,
    error: licenseGated ? undefined : error,
    licenseGated,
    onDraftJsonChange: setDraftJson,
    onValidate: onValidateCoalesced,
  };
}
