import { OpsShards } from '@/domains/ops/ops_shards';
import { useOpsShardsPageWorkspace } from '@/domains/ops/use_ops_shards_page_workspace';

export function OpsShardsPage() {
  return <OpsShards {...useOpsShardsPageWorkspace()} />;
}
