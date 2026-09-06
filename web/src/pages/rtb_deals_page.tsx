import { RtbDealsDirectory } from '@/domains/rtb/rtb_deals_directory';
import { useRtbDealsPageWorkspace } from '@/domains/rtb/use_rtb_deals_page_workspace';

export function RtbDealsPage() {
  return <RtbDealsDirectory {...useRtbDealsPageWorkspace()} />;
}
