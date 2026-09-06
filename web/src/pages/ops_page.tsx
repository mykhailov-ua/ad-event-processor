import { OpsHome } from '@/domains/ops/ops_home';
import { useOpsPageWorkspace } from '@/domains/ops/use_ops_page_workspace';

export function OpsPage() {
  return <OpsHome {...useOpsPageWorkspace()} />;
}
