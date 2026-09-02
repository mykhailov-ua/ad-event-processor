import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { PlatformCampaignLink, PlatformCampaignMutation } from '@/api/types';
import { IntegrationsNav, integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { PlatformCampaignLinkForm } from '@/domains/integrations/platform_campaign_link_form';
import { displayMicro, displayTimestamp } from '@/lib/display';

export type IntegrationsPlatformCampaignsProps = {
  links: PlatformCampaignLink[];
  appliedCustomerId: string;
  draftCustomerId: string;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  onDraftCustomerIdChange: (value: string) => void;
  onApplyCustomerScope: () => void;
  linkForm: {
    draftCampaignId: string;
    draftNetwork: string;
    draftExternalCampaignId: string;
    draftAccountId: string;
    draftDailyBudgetMicro: string;
    saving: boolean;
    deleting: boolean;
    refreshing: boolean;
    syncing: boolean;
    pausing: boolean;
    resuming: boolean;
    settingBudget: boolean;
    saveError: Error | undefined;
    deleteError: Error | undefined;
    refreshError: Error | undefined;
    syncError: Error | undefined;
    mutationError: Error | undefined;
    saveSuccess: boolean;
    deleteSuccess: boolean;
    refreshSuccess: boolean;
    syncSuccess: boolean;
    mutationResult: PlatformCampaignMutation | undefined;
    onDraftCampaignIdChange: (value: string) => void;
    onDraftNetworkChange: (value: string) => void;
    onDraftExternalCampaignIdChange: (value: string) => void;
    onDraftAccountIdChange: (value: string) => void;
    onDraftDailyBudgetMicroChange: (value: string) => void;
    onSave: () => void;
    onDelete: () => void;
    onRefresh: () => void;
    onSyncRun: () => void;
    onPause: () => void;
    onResume: () => void;
    onSetBudget: () => void;
    onPrefillFromLink: (row: PlatformCampaignLink) => void;
  };
};

export function IntegrationsPlatformCampaigns({
  links,
  appliedCustomerId,
  draftCustomerId,
  fetching,
  error,
  hasSnapshot,
  onDraftCustomerIdChange,
  onApplyCustomerScope,
  linkForm,
}: IntegrationsPlatformCampaignsProps) {
  if (!appliedCustomerId) {
    return (
      <PageChrome title="Platform campaign links">
        <IntegrationsNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list platform campaign links."
        />
      </PageChrome>
    );
  }

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Platform campaign links">
        <IntegrationsNav />
        <CustomerScopeBar
          appliedCustomerId={appliedCustomerId}
          draftCustomerId={draftCustomerId}
          onApply={onApplyCustomerScope}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
        />
        {integrationsPanelError(error, 'Could not load platform campaign links')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Platform campaign links">
      <IntegrationsNav />

      <CustomerScopeBar
        appliedCustomerId={appliedCustomerId}
        draftCustomerId={draftCustomerId}
        onApply={onApplyCustomerScope}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
      />

      <PlatformCampaignLinkForm
        disabled={!appliedCustomerId}
        draftCampaignId={linkForm.draftCampaignId}
        draftNetwork={linkForm.draftNetwork}
        draftExternalCampaignId={linkForm.draftExternalCampaignId}
        draftAccountId={linkForm.draftAccountId}
        draftDailyBudgetMicro={linkForm.draftDailyBudgetMicro}
        saving={linkForm.saving}
        deleting={linkForm.deleting}
        refreshing={linkForm.refreshing}
        syncing={linkForm.syncing}
        pausing={linkForm.pausing}
        resuming={linkForm.resuming}
        settingBudget={linkForm.settingBudget}
        saveError={linkForm.saveError}
        deleteError={linkForm.deleteError}
        refreshError={linkForm.refreshError}
        syncError={linkForm.syncError}
        mutationError={linkForm.mutationError}
        saveSuccess={linkForm.saveSuccess}
        deleteSuccess={linkForm.deleteSuccess}
        refreshSuccess={linkForm.refreshSuccess}
        syncSuccess={linkForm.syncSuccess}
        mutationResult={linkForm.mutationResult}
        onDraftCampaignIdChange={linkForm.onDraftCampaignIdChange}
        onDraftNetworkChange={linkForm.onDraftNetworkChange}
        onDraftExternalCampaignIdChange={linkForm.onDraftExternalCampaignIdChange}
        onDraftAccountIdChange={linkForm.onDraftAccountIdChange}
        onDraftDailyBudgetMicroChange={linkForm.onDraftDailyBudgetMicroChange}
        onSave={linkForm.onSave}
        onDelete={linkForm.onDelete}
        onRefresh={linkForm.onRefresh}
        onSyncRun={linkForm.onSyncRun}
        onPause={linkForm.onPause}
        onResume={linkForm.onResume}
        onSetBudget={linkForm.onSetBudget}
      />

      {links.length === 0 ? (
        <EmptyState
          title="No links"
          description="No external platform campaign links exist for this customer."
        />
      ) : (
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Campaign</TableHead>
                <TableHead>Network</TableHead>
                <TableHead>External ID</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Daily budget (micro)</TableHead>
                <TableHead>Last synced</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {links.map((row) => (
                <TableRow
                  key={`${row.campaign_id}-${row.network}`}
                  className="cursor-pointer"
                  onClick={() => linkForm.onPrefillFromLink(row)}
                >
                  <TableCell className="font-mono text-xs">{row.campaign_id}</TableCell>
                  <TableCell>{row.network}</TableCell>
                  <TableCell className="font-mono text-xs">{row.external_campaign_id}</TableCell>
                  <TableCell>
                    {row.sync_error ? (
                      <Badge variant="destructive">{row.external_status ?? 'error'}</Badge>
                    ) : (
                      <Badge variant="outline">{row.external_status ?? 'unknown'}</Badge>
                    )}
                  </TableCell>
                  <TableCell>{displayMicro(row.external_daily_budget_micro)}</TableCell>
                  <TableCell>{displayTimestamp(row.last_synced_at)}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {error && hasSnapshot ? integrationsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
