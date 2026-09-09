import { Link } from 'react-router-dom';
import type { OpsDomainRotationResponse, OpsTlsAllowedResponse } from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageWithLoad } from '@/domains/ops/ops_page_shell';

type DomainRotationHostRow = {
  hostname: string;
  role?: string;
  health_status?: string;
  ssl_status?: string;
  pool_domain_status?: string;
  dmr_campaign_count?: number;
  active_campaign_count?: number;
};

function readRotationHosts(rotation: OpsDomainRotationResponse): DomainRotationHostRow[] {
  const hosts = rotation.hosts;
  if (!Array.isArray(hosts)) {
    return [];
  }
  return hosts.flatMap((row) => {
    if (row == null || typeof row !== 'object') {
      return [];
    }
    const hostname = (row as { hostname?: unknown }).hostname;
    if (typeof hostname !== 'string' || hostname === '') {
      return [];
    }
    return [row as DomainRotationHostRow];
  });
}

export type OpsDomainsProps = {
  rotation: OpsDomainRotationResponse | undefined;
  tlsHost: OpsTlsAllowedResponse | undefined;
  draftHostname: string;
  fetchingRotation: boolean;
  fetchingTlsHost: boolean;
  rotationError: Error | undefined;
  tlsHostError: Error | undefined;
  hasRotationSnapshot: boolean;
  onDraftHostnameChange: (value: string) => void;
  onLoadRotation: () => void;
  onLookupTlsHost: () => void;
};

export function OpsDomains({
  rotation,
  tlsHost,
  draftHostname,
  fetchingRotation,
  fetchingTlsHost,
  rotationError,
  tlsHostError,
  hasRotationSnapshot,
  onDraftHostnameChange,
  onLoadRotation,
  onLookupTlsHost,
}: OpsDomainsProps) {
  const rotationHosts = rotation ? readRotationHosts(rotation) : [];

  return (
    <OpsPageWithLoad
      blockingErrorTitle="Could not load domain rotation"
      fetchState={{
        fetching: fetchingRotation,
        error: rotationError,
        hasSnapshot: hasRotationSnapshot,
      }}
      refreshErrorTitle="Domain rotation refresh failed"
      title="Domain ops"
      alerts={tlsHostError ? opsPanelError(tlsHostError, 'TLS allow check failed') : null}
      filters={
        <div >
          <Label htmlFor="ops-tls-hostname">Hostname</Label>
          <Input
            id="ops-tls-hostname"
            value={draftHostname}
            onChange={(event) => onDraftHostnameChange(event.target.value)}
          />
        </div>
      }
      actions={
        <>
          <OpsActionGroup label="Domain data">
            <Button
              disabled={fetchingRotation}
              loading={fetchingRotation}
              type="button"
              onClick={onLoadRotation}
            >
              Load rotation
            </Button>
          </OpsActionGroup>
          <OpsActionGroup label="TLS check">
            <Button
              disabled={fetchingTlsHost}
              loading={fetchingTlsHost}
              type="button"
              onClick={onLookupTlsHost}
            >
              Check TLS allowed
            </Button>
          </OpsActionGroup>
        </>
      }
    >
      <p >
        <Link  to="/domains">
          Open domains directory
        </Link>{' '}
        for bulk import, SSL setup, wildcard certs, and burn workflows.
      </p>

      {rotation ? (
        <div >
          <p >Domain rotation ({rotationHosts.length})</p>
          {rotationHosts.length > 0 ? (
            <ul >
              {rotationHosts.map((row) => (
                <li key={row.hostname}>
                  <span >{row.hostname}</span>
                  <span >
                    {row.role ?? 'role n/a'}
                    {row.health_status ? ` | health ${row.health_status}` : ''}
                    {row.ssl_status ? ` | SSL ${row.ssl_status}` : ''}
                    {row.pool_domain_status ? ` | pool ${row.pool_domain_status}` : ''}
                    {row.dmr_campaign_count != null ? ` | DMR ${row.dmr_campaign_count}` : ''}
                    {row.active_campaign_count != null
                      ? ` | active campaigns ${row.active_campaign_count}`
                      : ''}
                  </span>
                </li>
              ))}
            </ul>
          ) : (
            <p >No rotation hosts returned.</p>
          )}
        </div>
      ) : null}
      {tlsHost ? (
        <div >
          <span >
            TLS allowed for <span >{draftHostname.trim() || 'hostname'}</span>
          </span>
          <Badge variant="default">yes</Badge>
        </div>
      ) : null}
    </OpsPageWithLoad>
  );
}
