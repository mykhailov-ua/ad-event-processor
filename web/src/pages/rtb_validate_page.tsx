import { useCallback, useState } from 'react';

import { validateRtbBidRequest } from '@/api/rtb_api';
import type { OpenRtbValidationResult } from '@/api/types';
import { RtbValidateBidRequest } from '@/domains/rtb/rtb_validate_bid_request';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';

const DEFAULT_JSON = '{\n  "id": "test-request"\n}';

export function RtbValidatePage() {
  const [draftJson, setDraftJson] = useState(DEFAULT_JSON);
  const [result, setResult] = useState<OpenRtbValidationResult | undefined>(undefined);
  const [validating, setValidating] = useState(false);
  const [error, setError] = useState<Error | undefined>(undefined);
  const [licenseGated, setLicenseGated] = useState(false);

  const onValidate = useCallback(async () => {
    setValidating(true);
    setError(undefined);
    setLicenseGated(false);
    try {
      const parsed = JSON.parse(draftJson) as Record<string, unknown>;
      const next = await validateRtbBidRequest(parsed);
      setResult(next);
    } catch (err) {
      const wrapped = err instanceof Error ? err : new Error(String(err));
      setError(wrapped);
      setLicenseGated(rtbLicenseGated(wrapped));
    } finally {
      setValidating(false);
    }
  }, [draftJson]);

  return (
    <RtbValidateBidRequest
      draftJson={draftJson}
      result={result}
      validating={validating}
      error={licenseGated ? undefined : error}
      licenseGated={licenseGated}
      onDraftJsonChange={setDraftJson}
      onValidate={() => {
        void onValidate();
      }}
    />
  );
}
