import { useCallback, useState } from 'react';

import { getOpsRum } from '@/api/ops_api';
import { OpsRum } from '@/domains/ops/ops_rum';

export function OpsRumPage() {
  const [payload, setPayload] = useState<Record<string, unknown> | undefined>();
  const [fetching, setFetching] = useState(false);
  const [error, setError] = useState<Error | undefined>();
  const [hasSnapshot, setHasSnapshot] = useState(false);

  const onLoad = useCallback(async () => {
    setFetching(true);
    setError(undefined);
    try {
      setPayload(await getOpsRum());
      setHasSnapshot(true);
    } catch (err) {
      setError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setFetching(false);
    }
  }, []);

  return (
    <OpsRum
      payload={payload}
      fetching={fetching}
      error={error}
      hasSnapshot={hasSnapshot}
      onLoad={onLoad}
    />
  );
}
