import { RtbFloorsApplyPanel } from '@/domains/rtb/rtb_floors_apply';
import { useRtbFloorsPageWorkspace } from '@/domains/rtb/use_rtb_floors_page_workspace';

export function RtbFloorsPage() {
  return <RtbFloorsApplyPanel {...useRtbFloorsPageWorkspace()} />;
}
