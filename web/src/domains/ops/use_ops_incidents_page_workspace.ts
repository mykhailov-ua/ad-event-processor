// L3 incidents snapshot: single GET on mount.
import { getOpsIncidents } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';

export function useOpsIncidentsPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => getOpsIncidents(signal), []);

  return {
    snapshot: data,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
