import { useCallback, useMemo, useState } from 'react';
import { toast } from 'sonner';

import { fetchPostbacksSnapshot, retryPostbackDlq } from '@/api/integrations_api';
import { mutationError } from '@/lib/mutation_audit';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useOpsPostbacksDlqWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching, revalidating } = useResource(
    (signal) => fetchPostbacksSnapshot(signal),
    [refreshToken]
  );

  const [retryingId, setRetryingId] = useState<string | undefined>();
  const [retryError, setRetryError] = useState<Error | undefined>();

  const dlq = useMemo(() => data?.dlq ?? [], [data?.dlq]);
  const listBusy = fetching || retryingId != null;
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, listBusy);

  const onRetry = useCallback(
    async (rowId: string) => {
      if (!rowId || retryingId != null) {
        return;
      }
      setRetryingId(rowId);
      setRetryError(undefined);
      try {
        await retryPostbackDlq(rowId);
        toast.success('Postback DLQ entry retried');
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        setRetryError(mutationError(err));
      } finally {
        setRetryingId(undefined);
      }
    },
    [bumpRefreshCoalesced, retryingId]
  );

  return {
    dlq,
    fetching,
    listRevalidating: revalidating,
    error,
    hasSnapshot: data != null,
    retryingId,
    retryError,
    onRetry,
  };
}
