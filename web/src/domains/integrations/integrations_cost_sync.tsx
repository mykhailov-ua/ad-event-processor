import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import { DatePicker } from '@/components/ui/datetime_picker';
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
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
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
  const networksFetchState = {
    fetching: fetchingNetworks && panel === 'networks' && !hasNetworks && !networksError,
    error: panel === 'networks' ? networksError : undefined,
    hasSnapshot: hasNetworks || panel !== 'networks',
  };

  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load cost sync networks"
      fetchState={networksFetchState}
      title="Cost sync"
    >
      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <FilterPanel>
        <h2 >Run cost sync</h2>
        <p >
          Enqueue a manual sync for the applied customer. Network and date range are optional; dates
          default to yesterday UTC on the server.
        </p>
        <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
          <div >
            <Label htmlFor="cost-sync-run-network">Network (optional)</Label>
            <Select
              value={runSyncForm.draftNetwork || '__all__'}
              onValueChange={(value) =>
                runSyncForm.onDraftNetworkChange(value === '__all__' ? '' : value)
              }
              disabled={!appliedCustomerId || runSyncForm.running}
            >
              <SelectTrigger id="cost-sync-run-network">
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
          <div >
            <Label htmlFor="cost-sync-run-from">From (UTC)</Label>
            <DatePicker
              id="cost-sync-run-from"
              value={runSyncForm.draftFrom}
              disabled={!appliedCustomerId || runSyncForm.running}
              onChange={runSyncForm.onDraftFromChange}
            />
          </div>
          <div >
            <Label htmlFor="cost-sync-run-to">To (UTC)</Label>
            <DatePicker
              id="cost-sync-run-to"
              value={runSyncForm.draftTo}
              disabled={!appliedCustomerId || runSyncForm.running}
              onChange={runSyncForm.onDraftToChange}
            />
          </div>
          <Button
            disabled={!appliedCustomerId || runSyncForm.running}
            onClick={runSyncForm.onRun}
            type="button"
          >
            {runSyncForm.running ? 'Running...' : 'Run sync'}
          </Button>
        </DirectoryFilterForm>
        {runSyncForm.runError
          ? integrationsPanelError(runSyncForm.runError, 'Cost sync run failed')
          : null}
        {runSyncForm.runSuccess ? (
          <p >
            Sync accepted. Refresh history for results.
          </p>
        ) : null}
      </FilterPanel>

      <div >
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
        <section >
          <h2 >Networks</h2>
          {networks.length === 0 ? (
            <EmptyState
              title="No networks"
              description="Cost sync network schemas returned no entries."
            />
          ) : (
            <DirectoryTable>
              <TableHeader>
                <TableRow>
                  <DirectoryTableHead>Network</DirectoryTableHead>
                  <DirectoryTableHead>Label</DirectoryTableHead>
                  <DirectoryTableHead>Account field</DirectoryTableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {networks.map((row) => (
                  <TableRow key={row.network}>
                    <TableCell >{row.network}</TableCell>
                    <TableCell>{row.label}</TableCell>
                    <TableCell>{row.account_id_label ?? ''}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </DirectoryTable>
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
          <section >
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

            <div >
              <h2 >Credentials</h2>
              {credentials.length === 0 ? (
                <EmptyState
                  title="No credentials"
                  description="No cost sync credentials are stored for this customer."
                />
              ) : (
                <DirectoryTable>
                  <TableHeader>
                    <TableRow>
                      <DirectoryTableHead>Network</DirectoryTableHead>
                      <DirectoryTableHead>Account</DirectoryTableHead>
                      <DirectoryTableHead>Interval (min)</DirectoryTableHead>
                      <DirectoryTableHead>Updated</DirectoryTableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {credentials.map((row) => (
                      <TableRow
                        key={`${row.customer_id}-${row.network}`}
                       
                        onClick={() => credentialForm.onPrefillFromCredential(row)}
                      >
                        <TableCell >{row.network}</TableCell>
                        <TableCell>{row.account_id ?? ''}</TableCell>
                        <TableCell>{row.sync_interval_minutes}</TableCell>
                        <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </DirectoryTable>
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
          <section >
            <h2 >Sync history</h2>
            {history.length === 0 ? (
              <EmptyState
                title="No sync runs"
                description="No cost sync runs recorded for this customer."
              />
            ) : (
              <DirectoryTable>
                <TableHeader>
                  <TableRow>
                    <DirectoryTableHead>Run</DirectoryTableHead>
                    <DirectoryTableHead>Network</DirectoryTableHead>
                    <DirectoryTableHead>Date</DirectoryTableHead>
                    <DirectoryTableHead>Status</DirectoryTableHead>
                    <DirectoryTableHead>Rows</DirectoryTableHead>
                    <DirectoryTableHead>Amount (USD micro)</DirectoryTableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {history.map((row) => (
                    <TableRow key={row.id}>
                      <TableCell>{row.id}</TableCell>
                      <TableCell >{row.network}</TableCell>
                      <TableCell>{row.cost_date}</TableCell>
                      <TableCell>
                        <Badge variant="outline">{row.status}</Badge>
                      </TableCell>
                      <TableCell>{row.rows_imported}</TableCell>
                      <TableCell>{displayMicro(row.total_amount_usd_micro)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </DirectoryTable>
            )}
          </section>
        )
      ) : null}

      {scopedError && hasScopedData
        ? integrationsPanelError(scopedError, 'Refresh failed')
        : null}
    </IntegrationsPageWithLoad>
  );
}
