import { FlowsDirectory } from '@/domains/creative/flows_directory';
import { useFlowsPageWorkspace } from '@/domains/creative/use_flows_page_workspace';

export function FlowsPage() {
  return <FlowsDirectory {...useFlowsPageWorkspace()} />;
}
