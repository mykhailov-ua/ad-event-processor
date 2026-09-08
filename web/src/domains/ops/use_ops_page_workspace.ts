// ops home: snapshot refresh + side-effect actions (role reload, support bundle download).
import { useCallback, useState } from 'react';

import { fetchOpsHomeSnapshot, postOpsSupportBundle, reloadOpsRoles } from '@/api/ops_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

export function useOpsPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();
  const { data, error, fetching } = useResource(
    (signal) => fetchOpsHomeSnapshot(signal),
    [refreshToken]
  );

  const [reloadingRoles, setReloadingRoles] = useState(false);
  const [rolesReloadError, setRolesReloadError] = useState<Error | undefined>();
  const [rolesReloadMessage, setRolesReloadMessage] = useState<string | undefined>();
  const [downloadingBundle, setDownloadingBundle] = useState(false);
  const [bundleDownloadError, setBundleDownloadError] = useState<Error | undefined>();

  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || reloadingRoles);

  const onReloadRoles = useCallback(() => {
    if (reloadingRoles) {
      return;
    }
    setReloadingRoles(true);
    setRolesReloadError(undefined);
    setRolesReloadMessage(undefined);
    void reloadOpsRoles()
      .then((result) => {
        setRolesReloadMessage(result.status ?? 'ok');
        bumpRefreshCoalesced();
      })
      .catch((err: unknown) => {
        setRolesReloadError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setReloadingRoles(false);
      });
  }, [bumpRefreshCoalesced, reloadingRoles]);

  const onDownloadSupportBundle = useCallback(() => {
    setDownloadingBundle(true);
    setBundleDownloadError(undefined);
    void postOpsSupportBundle()
      .then((result) => {
        triggerBlobDownload(result.blob, result.filename);
      })
      .catch((err: unknown) => {
        setBundleDownloadError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setDownloadingBundle(false);
      });
  }, []);

  return {
    snapshot: data,
    fetching,
    error,
    hasSnapshot: data != null,
    reloadingRoles,
    rolesReloadError,
    rolesReloadMessage,
    onReloadRoles,
    downloadingBundle,
    bundleDownloadError,
    onDownloadSupportBundle,
  };
}
