// shard admin: list snapshot + shard-0 catchup mutation; refresh coalesced while catchup in flight.
import { useCallback, useState } from 'react';
import { toast } from 'sonner';

import { listOpsShards, triggerOpsShard0Catchup } from '@/api/ops_api';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';
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
    if (!confirmDestructiveAction('Run shard-0 config catch-up now?')) {
      return;
    }
    setCatchingUp(true);
    setCatchupError(undefined);
    try {
      const result = await triggerOpsShard0Catchup();
      setCatchupStatus(result.status ?? 'accepted');
      toast.success('Shard-0 catch-up accepted');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setCatchupError(mutationError(err));
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
