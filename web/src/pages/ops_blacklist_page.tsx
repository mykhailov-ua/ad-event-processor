import { OpsBlacklist } from '@/domains/ops/ops_blacklist';
import { useOpsBlacklistPageWorkspace } from '@/domains/ops/use_ops_blacklist_page_workspace';

export function OpsBlacklistPage() {
  return <OpsBlacklist {...useOpsBlacklistPageWorkspace()} />;
}
