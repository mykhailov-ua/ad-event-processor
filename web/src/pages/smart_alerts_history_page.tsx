import { useCallback, useMemo, useState } from 'react';

import { ackSmartAlertEvent, listSmartAlertHistory } from '@/api/smart_alerts_api';
import { SmartAlertsHistoryDirectory } from '@/domains/automation/smart_alerts_history_directory';
import { useCustomerScope } from '@/hooks/use_customer_scope';
import { useResource } from '@/hooks/use_resource';

export function SmartAlertsHistoryPage() {
  const {
    appliedCustomerId,
    draftCustomerId,
    setDraftCustomerId,
    applyCustomerScope,
  } = useCustomerScope();

  const shouldFetch = Boolean(appliedCustomerId);
  const [reloadKey, setReloadKey] = useState(0);
  const [ackingEventId, setAckingEventId] = useState<string | undefined>(undefined);
  const [ackError, setAckError] = useState<Error | undefined>(undefined);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!shouldFetch) {
        return Promise.resolve([]);
      }
      return listSmartAlertHistory({ customer_id: appliedCustomerId, limit: 50 }, signal);
    },
    [appliedCustomerId, reloadKey, shouldFetch],
  );

  const items = useMemo(() => data ?? [], [data]);

  const onAck = useCallback(async (eventId: string) => {
    setAckingEventId(eventId);
    setAckError(undefined);
    try {
      await ackSmartAlertEvent(eventId);
      setReloadKey((value) => value + 1);
    } catch (err) {
      setAckError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setAckingEventId(undefined);
    }
  }, []);

  return (
    <SmartAlertsHistoryDirectory
      items={items}
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      fetching={fetching}
      error={error}
      hasSnapshot={!shouldFetch || data != null}
      ackingEventId={ackingEventId}
      ackError={ackError}
      onDraftCustomerIdChange={setDraftCustomerId}
      onApplyCustomerScope={applyCustomerScope}
      onAck={(eventId) => {
        void onAck(eventId);
      }}
    />
  );
}
