// ops health radar: stack snapshot + ops home doctor checks + lazy domain probe.
import { useState } from 'react';

import { probeDomain } from '@/api/domains_api';
import { fetchOpsHomeSnapshot, getStackHealthSnapshot } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { requireNonEmpty } from '@/lib/admin_validation_error';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsHealthPageWorkspace() {
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const pageResource = useResource(
    async (signal) => {
      const [stackHealth, opsHome] = await Promise.all([
        getStackHealthSnapshot(signal),
        fetchOpsHomeSnapshot(signal),
      ]);
      return { stackHealth, doctor: opsHome.doctor };
    },
    [refreshToken]
  );

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

  const refreshing = pageResource.fetching;
  const onRefresh = useCoalescedBumpRefresh(bumpRefresh, refreshing);

  const onRunProbe = useCoalescedCallback(
    () => {
      const hostnameResult = requireNonEmpty(draftHostname, 'Hostname', 'hostname');
      if (!hostnameResult.ok) {
        setProbeValidationError(hostnameResult.error);
        return;
      }
      setProbeValidationError(undefined);
      setProbeHostname(hostnameResult.value);
      setProbeToken((value) => value + 1);
    },
    {
      inFlightGuard: true,
      inFlight: probeResource.fetching,
    }
  );

  return {
    stackHealth: pageResource.data?.stackHealth,
    doctor: pageResource.data?.doctor,
    doctorFetching: pageResource.fetching,
    doctorError: undefined,
    hasDoctorSnapshot: pageResource.data?.doctor != null,
    fetching: pageResource.fetching,
    error: pageResource.error,
    hasSnapshot: pageResource.data?.stackHealth != null,
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
