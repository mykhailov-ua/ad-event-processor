import { Link } from 'react-router-dom';
import type { OpsDomainRotationResponse, OpsTlsAllowedResponse } from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';

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

  if (fetchingRotation && !hasRotationSnapshot && !rotationError) {
    return <OpsPageLoading />;
  }

  return (
    <OpsPageShell
      filters={
        <div className="grid gap-2">
          <Label htmlFor="ops-tls-hostname">Hostname</Label>
          <Input
            id="ops-tls-hostname"
            value={draftHostname}
            onChange={(event) => onDraftHostnameChange(event.target.value)}
          />
        </div>
      }
      title="Domain ops"
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
      <p className="text-sm text-muted-foreground">
        <Link className="text-primary underline-offset-4 hover:underline" to="/domains">
          Open domains directory
        </Link>{' '}
        for bulk import, SSL setup, wildcard certs, and burn workflows.
      </p>

      {rotationError && !hasRotationSnapshot
        ? opsPanelError(rotationError, 'Could not load domain rotation')
        : null}

      {rotation ? (
        <div className="grid gap-2 rounded-md border p-3 text-sm">
          <p className="font-medium">Domain rotation ({rotationHosts.length})</p>
          {rotationHosts.length > 0 ? (
            <ul className="grid gap-2">
              {rotationHosts.map((row) => (
                <li key={row.hostname} className="grid gap-0.5">
                  <span className="text-xs">{row.hostname}</span>
                  <span className="text-muted-foreground">
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
            <p className="text-muted-foreground">No rotation hosts returned.</p>
          )}
        </div>
      ) : null}
      {tlsHostError ? opsPanelError(tlsHostError, 'TLS allow check failed') : null}
      {tlsHost ? (
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="font-medium">
            TLS allowed for <span className="text-xs">{draftHostname.trim() || 'hostname'}</span>
          </span>
          <Badge variant="default">yes</Badge>
        </div>
      ) : null}
    </OpsPageShell>
  );
}
