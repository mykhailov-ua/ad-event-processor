import { RtbValidateBidRequest } from '@/domains/rtb/rtb_validate_bid_request';
import { useRtbValidatePageWorkspace } from '@/domains/rtb/use_rtb_validate_page_workspace';

export function RtbValidatePage() {
  return <RtbValidateBidRequest {...useRtbValidatePageWorkspace()} />;
}
