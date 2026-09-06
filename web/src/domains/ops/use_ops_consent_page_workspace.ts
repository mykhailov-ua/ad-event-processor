// L3 consent proofs: single GET snapshot on mount (Cold directory).
import { getOpsConsentProofs } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';

export function useOpsConsentPageWorkspace() {
  const { data, error, fetching } = useResource((signal) => getOpsConsentProofs(signal), []);

  return {
    payload: data,
    fetching,
    error,
    hasSnapshot: data != null,
  };
}
