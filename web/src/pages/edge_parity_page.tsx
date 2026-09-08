import { EdgeParityDirectory } from '@/domains/reports/edge_parity_directory';
import { useEdgeParityPageWorkspace } from '@/domains/reports/use_edge_parity_page_workspace';

export function EdgeParityPage() {
  return <EdgeParityDirectory {...useEdgeParityPageWorkspace()} />;
}
