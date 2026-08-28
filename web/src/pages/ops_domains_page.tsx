import { useMemo } from 'react';
import {
  buildDomainRotationUrl,
  buildDomainTlsAllowedUrl,
  isApiStubError,
  type DomainRotation,
  type DomainTlsAllowed,
} from '../helpers/ops_api.js';
import { useResource } from '../helpers/use_resource.js';
import { OpsDomainsPanel } from '../ui/ops/ops_domains_panel.js';

export function OpsDomainsPage() {
  const rotationUrl = buildDomainRotationUrl();
  const {
    data: rotation,
    loading,
    error,
  } = useResource<DomainRotation>(rotationUrl);

  const firstHost = rotation?.hosts?.[0]?.hostname;
  const tlsUrl = firstHost ? buildDomainTlsAllowedUrl(firstHost) : null;
  const { data: tlsAllowed } = useResource<DomainTlsAllowed>(tlsUrl);

  const stub = useMemo(() => isApiStubError(error), [error]);

  return (
    <OpsDomainsPanel
      rotation={rotation}
      tlsAllowed={tlsAllowed}
      loading={loading}
      error={error}
      stub={stub}
    />
  );
}
