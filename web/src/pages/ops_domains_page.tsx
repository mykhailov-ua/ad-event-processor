import { OpsDomains } from '@/domains/ops/ops_domains';
import { useOpsDomainsPageWorkspace } from '@/domains/ops/use_ops_domains_page_workspace';

export function OpsDomainsPage() {
  return <OpsDomains {...useOpsDomainsPageWorkspace()} />;
}
