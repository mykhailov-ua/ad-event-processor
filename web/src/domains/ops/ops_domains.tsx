import { Button } from '@/components/ui/button';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { opsPanelError } from '@/domains/ops/ops_nav';
import { OpsActionGroup, OpsPageLoading, OpsPageShell } from '@/domains/ops/ops_page_shell';

export type OpsDomainsProps = {
  rotation: Record<string, unknown> | undefined;
  tlsAllowed: Record<string, unknown> | undefined;
  tlsHost: Record<string, unknown> | undefined;
  draftHostname: string;
  fetchingRotation: boolean;
  fetchingTlsList: boolean;
  fetchingTlsHost: boolean;
  rotationError: Error | undefined;
  tlsListError: Error | undefined;
  tlsHostError: Error | undefined;
  hasRotationSnapshot: boolean;
  hasTlsListSnapshot: boolean;
  onDraftHostnameChange: (value: string) => void;
  onLoadRotation: () => void;
  onLoadTlsList: () => void;
  onLookupTlsHost: () => void;
};

export function OpsDomains({
  rotation,
  tlsAllowed,
  tlsHost,
  draftHostname,
  fetchingRotation,
  fetchingTlsList,
  fetchingTlsHost,
  rotationError,
  tlsListError,
  tlsHostError,
  hasRotationSnapshot,
  hasTlsListSnapshot,
  onDraftHostnameChange,
  onLoadRotation,
  onLoadTlsList,
  onLookupTlsHost,
}: OpsDomainsProps) {
  if (fetchingRotation && !hasRotationSnapshot && !rotationError) {
    return <OpsPageLoading />;
  }

  return (
    <OpsPageShell
      filters={
        <label className="admin-label">
          Hostname
          <input
            className="admin-input"
            id="ops-tls-hostname"
            value={draftHostname}
            onChange={(event) => onDraftHostnameChange(event.target.value)}
          />
        </label>
      }
      title="Domain ops"
      actions={
        <>
          <OpsActionGroup label="Domain data">
            <Button disabled={fetchingRotation} loading={fetchingRotation} type="button" onClick={onLoadRotation}>
              Load rotation
            </Button>
            <Button disabled={fetchingTlsList} loading={fetchingTlsList} type="button" onClick={onLoadTlsList}>
              Load TLS allowed list
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
      {rotationError && !hasRotationSnapshot
        ? opsPanelError(rotationError, 'Could not load domain rotation')
        : null}
      {tlsListError && !hasTlsListSnapshot
        ? opsPanelError(tlsListError, 'Could not load TLS allowed list')
        : null}

      {rotation ? <JsonDashboardView payload={rotation} /> : null}
      {tlsAllowed ? <JsonDashboardView payload={tlsAllowed} /> : null}
      {tlsHostError ? opsPanelError(tlsHostError, 'TLS allow check failed') : null}
      {tlsHost ? <JsonDashboardView payload={tlsHost} /> : null}
    </OpsPageShell>
  );
}
