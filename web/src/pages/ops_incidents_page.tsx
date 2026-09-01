import { getOpsIncidents } from '@/api/ops_api';
import { OpsIncidents } from '@/domains/ops/ops_incidents';
import { useResource } from '@/hooks/use_resource';

export function OpsIncidentsPage() {
  const { data, error, fetching } = useResource((signal) => getOpsIncidents(signal), []);

  return (
    <OpsIncidents
      snapshot={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
    />
  );
}
