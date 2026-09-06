import { RtbIntegrationProfilePanel } from '@/domains/rtb/rtb_integration_profile';
import { useRtbIntegrationProfilePageWorkspace } from '@/domains/rtb/use_rtb_integration_profile_page_workspace';

export function RtbIntegrationProfilePage() {
  return <RtbIntegrationProfilePanel {...useRtbIntegrationProfilePageWorkspace()} />;
}
