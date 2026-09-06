import { OpsRecon } from '@/domains/ops/ops_recon';
import { useOpsReconPageWorkspace } from '@/domains/ops/use_ops_recon_page_workspace';

export function OpsReconPage() {
  return <OpsRecon {...useOpsReconPageWorkspace()} />;
}
