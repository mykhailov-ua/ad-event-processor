import { DomainsDirectory } from '@/domains/creative/domains_directory';
import { useDomainsPageWorkspace } from '@/domains/creative/use_domains_page_workspace';

export function DomainsPage() {
  return <DomainsDirectory {...useDomainsPageWorkspace()} />;
}
