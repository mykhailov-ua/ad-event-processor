import { useMemo, useState } from 'react';

import { PageChrome } from '@/shell/page_chrome';
import { CustomerScopeBar } from '@/shell/customer_scope_bar';
import { EmptyState } from '@/shell/empty_state';
import { Badge } from '@/components/ui/badge';
import type { PlatformCampaignLink, PlatformCampaignMutation } from '@/api/types';
import { IntegrationsNav, IntegrationsPageWithLoad } from '@/domains/integrations/integrations_nav';
import { PlatformCampaignLinkForm } from '@/domains/integrations/platform_campaign_link_form';
import { displayMicro, displayTimestamp } from '@/lib/display';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { adminTypography } from '@/lib/admin_kit';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryRecordMap,
  directoryOperateRows,
} from '@/shell/directory_select_overview_table';
import { TableHost } from '@/shell/ui_bands';

export type IntegrationsPlatformCampaignsProps = {
  links?: PlatformCampaignLink[];
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
    formValidationError?: AdminValidationError;
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

function platformLinkId(row: PlatformCampaignLink): string {
  return `${row.campaign_id}-${row.network}`;
}

function buildPlatformLinkOverviewFields(row: PlatformCampaignLink): DirectoryOverviewField[] {
  const statusLabel = row.sync_error
    ? (row.external_status ?? 'error')
    : (row.external_status ?? 'unknown');
  return [
    {
      label: 'Campaign',
      value: <span className={adminTypography.monoData}>{row.campaign_id}</span>,
    },
    { label: 'Network', value: row.network },
    {
      label: 'External ID',
      value: <span className={adminTypography.monoData}>{row.external_campaign_id}</span>,
    },
    {
      label: 'Status',
      value: (
        <Badge variant={row.sync_error ? 'destructive' : 'outline'}>{statusLabel}</Badge>
      ),
    },
    { label: 'Daily budget (micro)', value: displayMicro(row.external_daily_budget_micro) },
    { label: 'Last synced', value: displayTimestamp(row.last_synced_at) },
    { label: 'Account ID', value: row.account_id ?? '' },
    { label: 'Sync error', value: row.sync_error ?? '' },
  ];
}

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
  const [selectedId, setSelectedId] = useState<string | null>(null);

  const recordById = useMemo(
    () => directoryRecordMap(links, platformLinkId),
    [links]
  );
  const rows = useMemo(
    () =>
      directoryOperateRows(
        links,
        platformLinkId,
        (row) => row.external_campaign_id ?? platformLinkId(row)
      ),
    [links]
  );

  const handleSelectedIdChange = (id: string | null) => {
    setSelectedId(id);
    if (id) {
      const record = recordById.get(id);
      if (record) {
        linkForm.onPrefillFromLink(record);
      }
    }
  };

  const customerScopeBar = (
    <CustomerScopeBar
      appliedCustomerId={appliedCustomerId}
      draftCustomerId={draftCustomerId}
      onApply={onApplyCustomerScope}
      onDraftCustomerIdChange={onDraftCustomerIdChange}
    />
  );

  if (!appliedCustomerId) {
    return (
      <PageChrome title="Platform campaign links">
        <IntegrationsNav />
        {customerScopeBar}
        <EmptyState
          title="Customer required"
          description="Apply a customer ID to list platform campaign links."
        />
      </PageChrome>
    );
  }

  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load platform campaign links"
      fetchState={{ error, fetching, hasSnapshot }}
      header={customerScopeBar}
      title="Platform campaign links"
    >
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
        formValidationError={linkForm.formValidationError}
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

      {(links ?? []).length === 0 ? (
        <EmptyState
          title="No links"
          description="No external platform campaign links exist for this customer."
        />
      ) : (
        <TableHost>
          <DirectorySelectOverviewTable
            buildOverviewFields={buildPlatformLinkOverviewFields}
            disabled={fetching}
            overviewTitle={(row) => row.external_campaign_id ?? platformLinkId(row)}
            recordById={recordById}
            revalidating={fetching && hasSnapshot}
            rows={rows}
            selectedId={selectedId}
            onSelectedIdChange={handleSelectedIdChange}
          />
        </TableHost>
      )}
    </IntegrationsPageWithLoad>
  );
}
