import { FlowDetail } from '@/domains/creative/flow_detail';
import { useFlowDetailPageWorkspace } from '@/domains/creative/use_flow_detail_page_workspace';

export function FlowDetailPage() {
  return <FlowDetail {...useFlowDetailPageWorkspace()} />;
}
