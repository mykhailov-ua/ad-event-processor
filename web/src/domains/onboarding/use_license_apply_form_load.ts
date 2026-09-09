// license apply form: GET license status only when showStatus; bumpRefresh after successful apply in parent.
import { getLicenseStatus } from '@/api/platform_api';
import { useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useResource } from '@/api/use_resource';

export function useLicenseApplyFormLoad(showStatus: boolean) {
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!showStatus) {
        return Promise.resolve(undefined);
      }
      return getLicenseStatus(signal);
    },
    [showStatus, refreshToken]
  );

  const bumpStatusRefresh = bumpRefresh;

  return {
    licenseStatus: data,
    statusError: error,
    statusFetching: fetching,
    bumpStatusRefresh,
  };
}

export type LicenseApplyFormLoad = ReturnType<typeof useLicenseApplyFormLoad>;
