// TLS/domain ops: separate lazy GET lanes for rotation list and per-host TLS ask.
import { useState } from 'react';

import { checkOpsTlsAllowed, getOpsDomainRotation } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';
import { requireNonEmpty } from '@/lib/admin_validation_error';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsDomainsPageWorkspace() {
  const [draftHostname, setDraftHostname] = useState('');
  const [rotationLoadToken, setRotationLoadToken] = useState(0);
  const [tlsHostLoadToken, setTlsHostLoadToken] = useState(0);
  const [lookupHostname, setLookupHostname] = useState('');
  const [tlsHostValidationError, setTlsHostValidationError] = useState<Error | undefined>();

  const rotationResource = useResource(
    (signal) => {
      if (rotationLoadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsDomainRotation(signal);
    },
    [rotationLoadToken]
  );

  const tlsHostResource = useResource(
    (signal) => {
      if (tlsHostLoadToken === 0 || !lookupHostname) {
        return skipLazyFetch();
      }
      return checkOpsTlsAllowed(lookupHostname, signal);
    },
    [tlsHostLoadToken, lookupHostname]
  );

  const onLoadRotation = useCoalescedBumpRefresh(() => {
    setRotationLoadToken((value) => value + 1);
  }, rotationResource.fetching);

  const onLookupTlsHost = useCoalescedCallback(
    () => {
      const hostnameResult = requireNonEmpty(draftHostname, 'Hostname', 'hostname');
      if (!hostnameResult.ok) {
        setTlsHostValidationError(hostnameResult.error);
        return;
      }
      setTlsHostValidationError(undefined);
      setLookupHostname(hostnameResult.value);
      setTlsHostLoadToken((value) => value + 1);
    },
    {
      inFlightGuard: true,
      inFlight: tlsHostResource.fetching,
    }
  );

  return {
    rotation: rotationResource.data,
    tlsHost: tlsHostResource.data,
    draftHostname,
    fetchingRotation: rotationResource.fetching,
    fetchingTlsHost: tlsHostResource.fetching,
    rotationError: rotationResource.error,
    tlsHostError: tlsHostValidationError ?? tlsHostResource.error,
    hasRotationSnapshot: rotationResource.data != null,
    onDraftHostnameChange: setDraftHostname,
    onLoadRotation,
    onLookupTlsHost,
  };
}
