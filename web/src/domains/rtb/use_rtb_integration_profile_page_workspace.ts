// L3 RTB integration profile: single GET; licenseGated clears blocking error for StubBanner.
import { getRtbIntegrationProfile } from '@/api/rtb_api';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useResource } from '@/api/use_resource';

export function useRtbIntegrationProfilePageWorkspace() {
  const { data, error, fetching } = useResource((signal) => getRtbIntegrationProfile(signal), []);

  const licenseGated = rtbLicenseGated(error);

  return {
    profile: data,
    fetching,
    error: licenseGated ? undefined : error,
    hasSnapshot: data != null || licenseGated,
    licenseGated,
  };
}
