import { listFlows } from '@/api/flows_api';
import { useResource } from '@/api/use_resource';

export function useFlowsPageWorkspace() {
  const { data, error, fetching, revalidating } = useResource((signal) => listFlows(signal), []);

  return {
    flows: data,
    error,
    fetching,
    listRevalidating: revalidating,
    hasSnapshot: data != null,
  };
}

export type FlowsPageWorkspace = ReturnType<typeof useFlowsPageWorkspace>;
