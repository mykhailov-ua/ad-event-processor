// L3 shard admin: list snapshot + shard-0 catchup mutation; refresh coalesced while catchup in flight.
import { useCallback, useState } from 'react';

import { listOpsShards, triggerOpsShard0Catchup } from '@/api/ops_api';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useOpsShardsPageWorkspace() {
  const [catchingUp, setCatchingUp] = useState(false);
  const [catchupError, setCatchupError] = useState<Error | undefined>();
  const [catchupStatus, setCatchupStatus] = useState<string | undefined>();
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource((signal) => listOpsShards(signal), [refreshToken]);

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || catchingUp);

  const runCatchup = useCallback(async () => {
    setCatchingUp(true);
    setCatchupError(undefined);
    try {
      const result = await triggerOpsShard0Catchup();
      setCatchupStatus(result.status ?? 'accepted');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCatchupError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setCatchingUp(false);
    }
  }, [bumpRefreshCoalesced]);

  const onCatchup = useCoalescedCallback(
    () => {
      void runCatchup();
    },
    { inFlightGuard: true, inFlight: catchingUp }
  );

  return {
    snapshot: data,
    fetching,
    error,
    hasSnapshot: data != null,
    catchingUp,
    catchupError,
    catchupStatus,
    onCatchup,
  };
}
