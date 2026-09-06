import { RtbDealEditor } from '@/domains/rtb/rtb_deal_editor';
import { useRtbDealEditorPageWorkspace } from '@/domains/rtb/use_rtb_deal_editor_page_workspace';

export function RtbDealEditorPage() {
  return <RtbDealEditor {...useRtbDealEditorPageWorkspace()} />;
}
