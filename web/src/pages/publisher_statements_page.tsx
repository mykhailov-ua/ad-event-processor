import { listPublisherStatements } from '@/api/publisher_api';
import { PublisherStatementsPanel } from '@/domains/portals/publisher_statements_panel';
import { useResource } from '@/api/use_resource';

export function PublisherStatementsPage() {
  const { data, error, fetching } = useResource(
    (signal) => listPublisherStatements({ limit: 50 }, signal),
    [],
  );

  return (
    <PublisherStatementsPanel
      statements={data?.items}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null || Boolean(error)}
    />
  );
}
