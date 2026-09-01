import { PageChrome } from '@/components/system/page_chrome';
import { CustomerScopeBar } from '@/components/system/customer_scope_bar';
import { EmptyState } from '@/components/system/empty_state';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { ErrorBlock } from '@/components/system/error_block';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { CostSyncCredential, CostSyncNetworkSchema, CostSyncRun } from '@/api/types';
import { CostSyncCredentialForm } from '@/domains/integrations/cost_sync_credential_form';
import { IntegrationsNav, integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type IntegrationsCostSyncPanel = 'networks' | 'credentials' | 'history';

const COST_SYNC_PANELS: { id: IntegrationsCostSyncPanel; label: string }[] = [
  { id: 'networks', label: 'Networks' },
  { id: 'credentials', label: 'Credentials' },
  { id: 'history', label: 'Sync history' },
];

export type IntegrationsCostSyncProps = {
  panel: IntegrationsCostSyncPanel;
  onPanelChange: (panel: IntegrationsCostSyncPanel) => void;
  networks: CostSyncNetworkSchema[];
  credentials: CostSyncCredential[];
  history: CostSyncRun[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetchingNetworks: boolean;
  fetchingScoped: boolean;
  networksError: Error | undefined;
  scopedError: Error | undefined;
  hasNetworks: boolean;
  hasScopedData: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  runSyncForm: {
    draftNetwork: string;
    draftFrom: string;
    draftTo: string;
    running: boolean;
    runError: Error | undefined;
    runSuccess: boolean;
    onDraftNetworkChange: (value: string) => void;
    onDraftFromChange: (value: string) => void;
    onDraftToChange: (value: string) => void;
    onRun: () => void;
  };
  credentialForm: {
    draftNetwork: string;
    draftAccountId: string;
    draftAccessToken: string;
    draftRefreshToken: string;
    draftApiKey: string;
    draftSyncIntervalMinutes: string;
    saving: boolean;
    deleting: boolean;
    saveError: Error | undefined;
    deleteError: Error | undefined;
    saveSuccess: boolean;
    deleteSuccess: boolean;
    onDraftNetworkChange: (value: string) => void;
    onDraftAccountIdChange: (value: string) => void;
    onDraftAccessTokenChange: (value: string) => void;
    onDraftRefreshTokenChange: (value: string) => void;
    onDraftApiKeyChange: (value: string) => void;
    onDraftSyncIntervalMinutesChange: (value: string) => void;
    onSave: () => void;
    onDelete: () => void;
    onPrefillFromCredential: (row: CostSyncCredential) => void;
  };
};

export function IntegrationsCostSync({
  panel,
  onPanelChange,
  networks,
  credentials,
  history,
  appliedCustomerId,
  draftCustomerId,
  fetchingNetworks,
  fetchingScoped,
  networksError,
  scopedError,
  hasNetworks,
  hasScopedData,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  runSyncForm,
  credentialForm,
}: IntegrationsCostSyncProps) {
  if (fetchingNetworks && panel === 'networks' && !hasNetworks && !networksError) {
    return <PageSkeleton />;
  }

  if (networksError && panel === 'networks' && !hasNetworks) {
    return (
      <PageChrome title="Cost sync">
        <IntegrationsNav />
        {integrationsPanelError(networksError, 'Could not load cost sync networks')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Cost sync">
      <IntegrationsNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <section className="ui-filter-panel">
        <h2 className="text-base font-semibold">Run cost sync</h2>
        <p className="text-sm text-muted-foreground">
          Enqueue a manual sync for the applied customer. Network and date range are optional; dates
          default to yesterday UTC on the server.
        </p>
        <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
          <div className="grid gap-2">
            <Label htmlFor="cost-sync-run-network">Network (optional)</Label>
            <Select
              value={runSyncForm.draftNetwork || '__all__'}
              onValueChange={(value) =>
                runSyncForm.onDraftNetworkChange(value === '__all__' ? '' : value)
              }
              disabled={!appliedCustomerId || runSyncForm.running}
            >
              <SelectTrigger id="cost-sync-run-network" className="h-9 w-full text-sm">
                <SelectValue placeholder="All networks" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__all__">All networks</SelectItem>
                {networks.map((row) => (
                  <SelectItem key={row.network} value={row.network ?? ''}>
                    {row.label ?? row.network}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="cost-sync-run-from">From (UTC)</Label>
            <Input
              id="cost-sync-run-from"
              type="date"
              value={runSyncForm.draftFrom}
              disabled={!appliedCustomerId || runSyncForm.running}
              onChange={(event) => runSyncForm.onDraftFromChange(event.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="cost-sync-run-to">To (UTC)</Label>
            <Input
              id="cost-sync-run-to"
              type="date"
              value={runSyncForm.draftTo}
              disabled={!appliedCustomerId || runSyncForm.running}
              onChange={(event) => runSyncForm.onDraftToChange(event.target.value)}
            />
          </div>
          <Button
            disabled={!appliedCustomerId || runSyncForm.running}
            onClick={runSyncForm.onRun}
            type="button"
          >
            {runSyncForm.running ? 'Running...' : 'Run sync'}
          </Button>
        </div>
        {runSyncForm.runError ? (
          <ErrorBlock title="Cost sync run failed" message={runSyncForm.runError.message} />
        ) : null}
        {runSyncForm.runSuccess ? (
          <p className="text-sm text-muted-foreground">Sync accepted. Refresh history for results.</p>
        ) : null}
      </section>

      <div className="flex flex-wrap gap-2">
        {COST_SYNC_PANELS.map((item) => (
          <Button
            key={item.id}
            type="button"
            variant={panel === item.id ? 'default' : 'outline'}
            onClick={() => onPanelChange(item.id)}
          >
            {item.label}
          </Button>
        ))}
      </div>

      {panel === 'networks' ? (
      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Networks</h2>
        {networks.length === 0 ? (
          <EmptyState
            title="No networks"
            description="Cost sync network schemas returned no entries."
          />
        ) : (
          <div className="ui-table-frame">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Network</TableHead>
                  <TableHead>Label</TableHead>
                  <TableHead>Account field</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {networks.map((row) => (
                  <TableRow key={row.network}>
                    <TableCell className="font-mono text-xs">{row.network}</TableCell>
                    <TableCell>{row.label}</TableCell>
                    <TableCell>{row.account_id_label ?? ''}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </section>
      ) : null}

      {panel === 'credentials' ? (
        !appliedCustomerId ? (
          <EmptyState
            title="Customer required"
            description="Apply a customer ID to load cost sync credentials."
          />
        ) : fetchingScoped && !hasScopedData && !scopedError ? (
          <PageSkeleton />
        ) : scopedError && !hasScopedData ? (
          integrationsPanelError(scopedError, 'Could not load cost sync credentials')
        ) : (
          <section className="grid gap-4">
            <CostSyncCredentialForm
              networks={networks}
              disabled={!appliedCustomerId}
              draftNetwork={credentialForm.draftNetwork}
              draftAccountId={credentialForm.draftAccountId}
              draftAccessToken={credentialForm.draftAccessToken}
              draftRefreshToken={credentialForm.draftRefreshToken}
              draftApiKey={credentialForm.draftApiKey}
              draftSyncIntervalMinutes={credentialForm.draftSyncIntervalMinutes}
              saving={credentialForm.saving}
              deleting={credentialForm.deleting}
              saveError={credentialForm.saveError}
              deleteError={credentialForm.deleteError}
              saveSuccess={credentialForm.saveSuccess}
              deleteSuccess={credentialForm.deleteSuccess}
              onDraftNetworkChange={credentialForm.onDraftNetworkChange}
              onDraftAccountIdChange={credentialForm.onDraftAccountIdChange}
              onDraftAccessTokenChange={credentialForm.onDraftAccessTokenChange}
              onDraftRefreshTokenChange={credentialForm.onDraftRefreshTokenChange}
              onDraftApiKeyChange={credentialForm.onDraftApiKeyChange}
              onDraftSyncIntervalMinutesChange={credentialForm.onDraftSyncIntervalMinutesChange}
              onSave={credentialForm.onSave}
              onDelete={credentialForm.onDelete}
            />

            <div className="grid gap-2">
            <h2 className="text-base font-semibold">Credentials</h2>
            {credentials.length === 0 ? (
              <EmptyState
                title="No credentials"
                description="No cost sync credentials are stored for this customer."
              />
            ) : (
              <div className="ui-table-frame">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Network</TableHead>
                      <TableHead>Account</TableHead>
                      <TableHead>Interval (min)</TableHead>
                      <TableHead>Updated</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {credentials.map((row) => (
                      <TableRow
                        key={`${row.customer_id}-${row.network}`}
                        className="cursor-pointer"
                        onClick={() => credentialForm.onPrefillFromCredential(row)}
                      >
                        <TableCell className="font-mono text-xs">{row.network}</TableCell>
                        <TableCell>{row.account_id ?? ''}</TableCell>
                        <TableCell>{row.sync_interval_minutes}</TableCell>
                        <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
            </div>
          </section>
        )
      ) : null}

      {panel === 'history' ? (
        !appliedCustomerId ? (
          <EmptyState
            title="Customer required"
            description="Apply a customer ID to load sync history."
          />
        ) : fetchingScoped && !hasScopedData && !scopedError ? (
          <PageSkeleton />
        ) : scopedError && !hasScopedData ? (
          integrationsPanelError(scopedError, 'Could not load cost sync history')
        ) : (
          <section className="grid gap-2">
            <h2 className="text-base font-semibold">Sync history</h2>
            {history.length === 0 ? (
              <EmptyState
                title="No sync runs"
                description="No cost sync runs recorded for this customer."
              />
            ) : (
              <div className="ui-table-frame">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Run</TableHead>
                      <TableHead>Network</TableHead>
                      <TableHead>Date</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Rows</TableHead>
                      <TableHead>Amount (USD micro)</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {history.map((row) => (
                      <TableRow key={row.id}>
                        <TableCell>{row.id}</TableCell>
                        <TableCell className="font-mono text-xs">{row.network}</TableCell>
                        <TableCell>{row.cost_date}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{row.status}</Badge>
                        </TableCell>
                        <TableCell>{row.rows_imported}</TableCell>
                        <TableCell>{displayMicro(row.total_amount_usd_micro)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </section>
        )
      ) : null}

      {scopedError && hasScopedData ? integrationsPanelError(scopedError, 'Refresh failed') : null}
    </PageChrome>
  );
}
