// publisher statements: paginated list (limit 50 fixed).
import { listPublisherStatements } from '@/api/publisher_api';
import { useResource } from '@/api/use_resource';

export function usePublisherStatementsPageWorkspace() {
  const { data, error, fetching } = useResource(
    (signal) => listPublisherStatements({ limit: 50 }, signal),
    []
  );

  return {
    statements: data?.items,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
