import { OpsRum } from '@/domains/ops/ops_rum';
import { useOpsRumPageWorkspace } from '@/domains/ops/use_ops_rum_page_workspace';

export function OpsRumPage() {
  return <OpsRum {...useOpsRumPageWorkspace()} />;
}
