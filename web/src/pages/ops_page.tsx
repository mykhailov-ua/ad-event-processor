import { useCallback, useState } from 'react';

import { fetchOpsHomeSnapshot, postOpsSupportBundle, reloadOpsRoles } from '@/api/ops_api';
import { OpsHome } from '@/domains/ops/ops_home';
import { useResource } from '@/hooks/use_resource';
import { triggerBlobDownload } from '@/lib/trigger_blob_download';

export function OpsPage() {
  const [reloadToken, setReloadToken] = useState(0);
  const { data, error, fetching } = useResource(
    (signal) => fetchOpsHomeSnapshot(signal),
    [reloadToken],
  );

  const [reloadingRoles, setReloadingRoles] = useState(false);
  const [rolesReloadError, setRolesReloadError] = useState<Error | undefined>();
  const [rolesReloadMessage, setRolesReloadMessage] = useState<string | undefined>();
  const [downloadingBundle, setDownloadingBundle] = useState(false);
  const [bundleDownloadError, setBundleDownloadError] = useState<Error | undefined>();

  const onReloadRoles = useCallback(() => {
    setReloadingRoles(true);
    setRolesReloadError(undefined);
    setRolesReloadMessage(undefined);
    void reloadOpsRoles()
      .then((result) => {
        setRolesReloadMessage(result.status ?? 'ok');
        setReloadToken((value) => value + 1);
      })
      .catch((err: unknown) => {
        setRolesReloadError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setReloadingRoles(false);
      });
  }, []);

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

  return (
    <OpsHome
      snapshot={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      reloadingRoles={reloadingRoles}
      rolesReloadError={rolesReloadError}
      rolesReloadMessage={rolesReloadMessage}
      onReloadRoles={onReloadRoles}
      downloadingBundle={downloadingBundle}
      bundleDownloadError={bundleDownloadError}
      onDownloadSupportBundle={onDownloadSupportBundle}
    />
  );
}
