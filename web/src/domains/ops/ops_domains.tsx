import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { OpsNav, opsPanelError } from '@/domains/ops/ops_nav';

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
    return <PageSkeleton />;
  }

  return (
    <PageChrome title="Domain ops">
      <OpsNav />

      <div className="flex flex-wrap gap-2">
        <SecondaryActionButton disabled={fetchingRotation} loading={fetchingRotation} onClick={onLoadRotation} type="button">
          Load rotation
        </SecondaryActionButton>
        <SecondaryActionButton disabled={fetchingTlsList} loading={fetchingTlsList} onClick={onLoadTlsList} type="button">
          Load TLS allowed list
        </SecondaryActionButton>
      </div>

      {rotationError && !hasRotationSnapshot
        ? opsPanelError(rotationError, 'Could not load domain rotation')
        : null}
      {tlsListError && !hasTlsListSnapshot
        ? opsPanelError(tlsListError, 'Could not load TLS allowed list')
        : null}

      {rotation ? <JsonDashboardView payload={rotation} /> : null}
      {tlsAllowed ? <JsonDashboardView payload={tlsAllowed} /> : null}

      <div className="grid max-w-md grid-cols-[1fr_auto] items-end gap-4">
        <div className="grid gap-2">
          <Label htmlFor="ops-tls-hostname">Hostname</Label>
          <Input
            id="ops-tls-hostname"
            value={draftHostname}
            onChange={(event) => onDraftHostnameChange(event.target.value)}
          />
        </div>
        <PrimaryActionButton disabled={fetchingTlsHost} loading={fetchingTlsHost} onClick={onLookupTlsHost} type="button">
          Check TLS allowed
        </PrimaryActionButton>
      </div>

      {tlsHostError ? opsPanelError(tlsHostError, 'TLS allow check failed') : null}
      {tlsHost ? <JsonDashboardView payload={tlsHost} /> : null}
    </PageChrome>
  );
}
