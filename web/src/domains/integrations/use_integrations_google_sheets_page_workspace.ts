import { useCallback, useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  disconnectGoogleSheets,
  getGoogleSheetsStatus,
  GOOGLE_SHEETS_CONNECT_PATH,
} from '@/api/integrations_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { mutationError } from '@/lib/mutation_audit';

export function useIntegrationsGoogleSheetsPageWorkspace() {
  const [searchParams, setSearchParams] = useSearchParams();
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data: status, error, fetching } = useResource(
    (signal) => getGoogleSheetsStatus(signal),
    [refreshToken]
  );
  const [disconnecting, setDisconnecting] = useState(false);
  const [disconnectError, setDisconnectError] = useState<Error | undefined>();
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || disconnecting);

  useEffect(() => {
    if (searchParams.get('google_sheets') !== 'connected') {
      return;
    }
    toast.success('Google Sheets connected');
    const next = new URLSearchParams(searchParams);
    next.delete('google_sheets');
    setSearchParams(next, { replace: true });
    bumpRefreshCoalesced();
  }, [bumpRefreshCoalesced, searchParams, setSearchParams]);

  const onConnect = useCallback(() => {
    window.location.assign(GOOGLE_SHEETS_CONNECT_PATH);
  }, []);

  const onDisconnect = useCallback(async () => {
    setDisconnecting(true);
    setDisconnectError(undefined);
    try {
      await disconnectGoogleSheets();
      toast.success('Google Sheets disconnected');
      bumpRefreshCoalesced();
    } catch (err: unknown) {
      setDisconnectError(mutationError(err));
    } finally {
      setDisconnecting(false);
    }
  }, [bumpRefreshCoalesced]);

  return {
    status,
    fetching,
    error,
    hasSnapshot: status != null,
    disconnecting,
    disconnectError,
    onConnect,
    onDisconnect,
  };
}
