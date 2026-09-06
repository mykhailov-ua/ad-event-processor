// L3 smart alert history: customer-scoped list + per-event ack mutation.
import { useCallback, useState } from 'react';

import { ackSmartAlertEvent, listSmartAlertHistory } from '@/api/smart_alerts_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/api/use_resource';

export function useSmartAlertsHistoryPageWorkspace() {
  const { appliedCustomerId, draftCustomerId, setDraftCustomerId, applyCustomerScope } =
    useCustomerScope();

  const shouldFetch = Boolean(appliedCustomerId);
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const [ackingEventId, setAckingEventId] = useState<string | undefined>(undefined);
  const [ackError, setAckError] = useState<Error | undefined>(undefined);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve(undefined);
      }
      return listSmartAlertHistory({ customer_id: appliedCustomerId, limit: 50 }, signal);
    },
    [appliedCustomerId, refreshToken, shouldFetch]
  );

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(
    bumpRefresh,
    fetching || ackingEventId != null
  );

  const onAck = useCallback(
    async (eventId: string) => {
      setAckingEventId(eventId);
      setAckError(undefined);
      try {
        await ackSmartAlertEvent(eventId);
        bumpRefreshCoalesced();
      } catch (err: unknown) {
        setAckError(err instanceof Error ? err : new Error(String(err)));
      } finally {
        setAckingEventId(undefined);
      }
    },
    [bumpRefreshCoalesced]
  );

  return {
    items: data,
    appliedCustomerId,
    draftCustomerId,
    fetching,
    error,
    hasSnapshot: data != null,
    ackingEventId,
    ackError,
    onDraftCustomerIdChange: setDraftCustomerId,
    onApplyCustomerScope: applyCustomerScope,
    onAck: (eventId: string) => {
      void onAck(eventId);
    },
  };
}
