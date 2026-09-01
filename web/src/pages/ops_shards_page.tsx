import { useCallback, useState } from 'react';

import { listOpsShards, triggerOpsShard0Catchup } from '@/api/ops_api';
import { OpsShards } from '@/domains/ops/ops_shards';
import { useResource } from '@/hooks/use_resource';

export function OpsShardsPage() {
  const [catchingUp, setCatchingUp] = useState(false);
  const [catchupError, setCatchupError] = useState<Error | undefined>();
  const [catchupStatus, setCatchupStatus] = useState<string | undefined>();
  const [refreshToken, setRefreshToken] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => listOpsShards(signal),
    [refreshToken],
  );

  const onCatchup = useCallback(async () => {
    setCatchingUp(true);
    setCatchupError(undefined);
    try {
      const result = await triggerOpsShard0Catchup();
      setCatchupStatus(result.status ?? 'accepted');
      setRefreshToken((value) => value + 1);
    } catch (err) {
      setCatchupError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCatchingUp(false);
    }
  }, []);

  return (
    <OpsShards
      snapshot={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      catchingUp={catchingUp}
      catchupError={catchupError}
      catchupStatus={catchupStatus}
      onCatchup={onCatchup}
    />
  );
}
