import { RtbOverview } from '@/domains/rtb/rtb_overview';
import { useRtbPageWorkspace } from '@/domains/rtb/use_rtb_page_workspace';

export function RtbPage() {
  return <RtbOverview {...useRtbPageWorkspace()} />;
}
