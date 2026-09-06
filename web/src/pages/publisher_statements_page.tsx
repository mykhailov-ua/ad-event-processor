import { PublisherStatementsPanel } from '@/domains/portals/publisher_statements_panel';
import { usePublisherStatementsPageWorkspace } from '@/domains/portals/use_publisher_statements_page_workspace';

export function PublisherStatementsPage() {
  return <PublisherStatementsPanel {...usePublisherStatementsPageWorkspace()} />;
}
