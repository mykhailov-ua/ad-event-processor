// L3 TLS/domain ops: separate lazy GET lanes for rotation list, allowed list, and per-host lookup.
import { useState } from 'react';

import { getOpsDomainRotation, getOpsTlsAllowedHost, getOpsTlsAllowedList } from '@/api/ops_api';
import { useResource } from '@/api/use_resource';
import { useCoalescedCallback } from '@/hooks/use_coalesced_callback';
import { useCoalescedBumpRefresh } from '@/hooks/use_coalesced_refresh_token';

function skipLazyFetch(): Promise<never> {
  return Promise.reject(new DOMException('Skipped', 'AbortError'));
}

export function useOpsDomainsPageWorkspace() {
  const [draftHostname, setDraftHostname] = useState('');
  const [rotationLoadToken, setRotationLoadToken] = useState(0);
  const [tlsListLoadToken, setTlsListLoadToken] = useState(0);
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

  const tlsListResource = useResource(
    (signal) => {
      if (tlsListLoadToken === 0) {
        return skipLazyFetch();
      }
      return getOpsTlsAllowedList(signal);
    },
    [tlsListLoadToken]
  );

  const tlsHostResource = useResource(
    (signal) => {
      if (tlsHostLoadToken === 0 || !lookupHostname) {
        return skipLazyFetch();
      }
      return getOpsTlsAllowedHost(lookupHostname, signal);
    },
    [tlsHostLoadToken, lookupHostname]
  );

  const onLoadRotation = useCoalescedBumpRefresh(() => {
    setRotationLoadToken((value) => value + 1);
  }, rotationResource.fetching);

  const onLoadTlsList = useCoalescedBumpRefresh(() => {
    setTlsListLoadToken((value) => value + 1);
  }, tlsListResource.fetching);

  const onLookupTlsHost = useCoalescedCallback(
    () => {
      const hostname = draftHostname.trim();
      if (!hostname) {
        setTlsHostValidationError(new Error('Hostname is required.'));
        return;
      }
      setTlsHostValidationError(undefined);
      setLookupHostname(hostname);
      setTlsHostLoadToken((value) => value + 1);
    },
    {
      inFlightGuard: true,
      inFlight: tlsHostResource.fetching,
    }
  );

  return {
    rotation: rotationResource.data,
    tlsAllowed: tlsListResource.data,
    tlsHost: tlsHostResource.data,
    draftHostname,
    fetchingRotation: rotationResource.fetching,
    fetchingTlsList: tlsListResource.fetching,
    fetchingTlsHost: tlsHostResource.fetching,
    rotationError: rotationResource.error,
    tlsListError: tlsListResource.error,
    tlsHostError: tlsHostValidationError ?? tlsHostResource.error,
    hasRotationSnapshot: rotationResource.data != null,
    hasTlsListSnapshot: tlsListResource.data != null,
    onDraftHostnameChange: setDraftHostname,
    onLoadRotation,
    onLoadTlsList,
    onLookupTlsHost,
  };
}
