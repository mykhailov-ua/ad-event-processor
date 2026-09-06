import { OpsIncidents } from '@/domains/ops/ops_incidents';
import { useOpsIncidentsPageWorkspace } from '@/domains/ops/use_ops_incidents_page_workspace';

export function OpsIncidentsPage() {
  return <OpsIncidents {...useOpsIncidentsPageWorkspace()} />;
}
