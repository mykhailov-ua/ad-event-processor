import { OpsHealth } from '@/domains/ops/ops_health';
import { useOpsHealthPageWorkspace } from '@/domains/ops/use_ops_health_page_workspace';

export function OpsHealthPage() {
  return <OpsHealth {...useOpsHealthPageWorkspace()} />;
}
