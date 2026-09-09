// ops health radar: stack snapshot + ops home doctor checks + lazy domain probe.
import { useState } from 'react';

import { probeDomain } from '@/api/domains_api';
import { fetchOpsHomeSnapshot, getStackHealthSnapshot } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsHealthPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const stackHealthResource = useResource(
    (signal) => getStackHealthSnapshot(signal),
    [refreshToken]
  );

  const opsHomeResource = useResource((signal) => fetchOpsHomeSnapshot(signal), [refreshToken]);

  const [draftHostname, setDraftHostname] = useState('');
  const [probeHostname, setProbeHostname] = useState('');
  const [probeToken, setProbeToken] = useState(0);
  const [probeValidationError, setProbeValidationError] = useState<Error | undefined>();

  const probeResource = useResource(
    (signal) => {
      if (probeToken === 0 || !probeHostname) {
        return skipLazyFetch();
      }
      return probeDomain(probeHostname, signal);
    },
    [probeToken, probeHostname]
  );

  const refreshing = stackHealthResource.fetching || opsHomeResource.fetching;
  const onRefresh = useCoalescedBumpRefresh(bumpRefresh, refreshing);

  const onRunProbe = useCoalescedCallback(
    () => {
      const hostname = draftHostname.trim();
      if (!hostname) {
        setProbeValidationError(new Error('Hostname is required.'));
        return;
      }
      setProbeValidationError(undefined);
      setProbeHostname(hostname);
      setProbeToken((value) => value + 1);
    },
    {
      inFlightGuard: true,
      inFlight: probeResource.fetching,
    }
  );

  return {
    stackHealth: stackHealthResource.data,
    doctor: opsHomeResource.data?.doctor,
    doctorFetching: opsHomeResource.fetching,
    doctorError: opsHomeResource.error,
    hasDoctorSnapshot: opsHomeResource.data?.doctor != null,
    fetching: stackHealthResource.fetching,
    error: stackHealthResource.error,
    hasSnapshot: stackHealthResource.data != null,
    refreshing,
    onRefresh,
    draftHostname,
    onDraftHostnameChange: setDraftHostname,
    probeResult: probeResource.data,
    probing: probeResource.fetching,
    probeError: probeValidationError ?? probeResource.error,
    onRunProbe,
  };
}
