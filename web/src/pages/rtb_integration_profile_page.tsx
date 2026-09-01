import { getRtbIntegrationProfile } from '@/api/rtb_api';
import { RtbIntegrationProfilePanel } from '@/domains/rtb/rtb_integration_profile';
import { rtbLicenseGated } from '@/domains/rtb/rtb_nav';
import { useResource } from '@/hooks/use_resource';

export function RtbIntegrationProfilePage() {
  const { data, error, fetching } = useResource(
    (signal) => getRtbIntegrationProfile(signal),
    [],
  );

  const licenseGated = rtbLicenseGated(error);

  return (
    <RtbIntegrationProfilePanel
      profile={data}
      fetching={fetching}
      error={licenseGated ? undefined : error}
      hasSnapshot={data != null || licenseGated}
      licenseGated={licenseGated}
    />
  );
}
