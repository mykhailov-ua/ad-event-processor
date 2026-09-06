import { RtbShadowTools } from '@/domains/rtb/rtb_shadow_tools';
import { useRtbShadowPageWorkspace } from '@/domains/rtb/use_rtb_shadow_page_workspace';

export function RtbShadowPage() {
  return <RtbShadowTools {...useRtbShadowPageWorkspace()} />;
}
