import { useCallback, useEffect, useMemo, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';

import { CampaignsListToolbar } from '@/domains/campaigns/list/campaigns_list_toolbar';
import { useCampaignsCommandPaletteActions } from '@/domains/campaigns/list/use_campaigns_command_palette_actions';
import { CampaignsDirectoryOverlays } from '@/domains/campaigns/list/campaigns_directory_overlays';
import { CampaignsSelectionPanel } from '@/domains/campaigns/list/campaigns_selection_panel';
import type { CampaignsDirectoryProps } from '@/domains/campaigns/list/campaigns_directory_types';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { StatusBadge } from '@/shell/status_badge';
import { campaignStatusToAdminTone } from '@/lib/admin_kit';
import { formatCampaignStatusLabel } from '@/lib/admin_typography';
import { useCampaignsDirectoryWorkspace } from '@/domains/campaigns/list/use_campaigns_directory_workspace';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { DirectoryFetchError, DirectoryPageShell } from '@/shell/directory_page_shell';
import { StubBanner } from '@/shell/stub_banner';
import { CAMPAIGN_LIST_FACETS_DEGRADED_MESSAGE } from '@/domains/campaigns/list/campaign_list_facets_source';
import { isCampaignListAuxEndpointUnavailable } from '@/domains/campaigns/list/campaign_list_aux_error';
import { runCampaignListBulkAction } from '@/domains/campaigns/list/campaign_list_bulk_guard';
import {
  openCampaignCreateDialog,
  openCampaignWizardSheet,
  setCampaignCreateDialogOpen,
  setCampaignWizardSheetOpen,
} from '@/domains/campaigns/list/campaign_list_create_overlay';
import { buildExportHubHref } from '@/lib/export_hub_paths';
import { rememberExportHubReturnPath } from '@/lib/export_hub_return';
import { listPageRange } from '@/lib/list_page_stats';
import { InAppLink } from '@/shell/in_app_link';
import { CampaignListNameWithSignals } from '@/domains/campaigns/list/campaign_list_operational_signals';
import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';
import {
  actionGuardError,
  toastValidationError,
  type AdminValidationError,
} from '@/lib/admin_validation_error';
import { ValidationErrorBlock } from '@/shell/validation_error_block';

function buildCampaignOverviewFields(
  campaign: Campaign,
  customerNameById: Record<string, string>,
  metrics?: CampaignListMetrics
): DirectoryOverviewField[] {
  const fields: DirectoryOverviewField[] = [
    { label: 'Name', value: campaign.name ?? campaign.id },
    { label: 'ID', value: campaign.id ?? '-' },
    {
      label: 'Status',
      value: (
        <StatusBadge
          label={formatCampaignStatusLabel(campaign.status)}
          tone={campaignStatusToAdminTone(campaign.status)}
        />
      ),
    },
    {
      label: 'Customer',
      value: customerNameById[campaign.customer_id ?? ''] ?? campaign.customer_id ?? '-',
    },
    { label: 'Budget limit', value: campaign.budget_limit ?? '-' },
    { label: 'Pacing', value: metrics?.pacing_mode ?? campaign.pacing_mode ?? '-' },
  ];
  if (metrics?.budget_burn_pct != null && Number.isFinite(metrics.budget_burn_pct)) {
    fields.push({ label: 'Budget burn', value: `${metrics.budget_burn_pct.toFixed(1)}%` });
  }
  if (metrics?.pacing_health) {
    fields.push({ label: 'Pacing health', value: metrics.pacing_health });
  }
  if (metrics?.metrics_stale) {
    fields.push({ label: 'Metrics freshness', value: 'Stale' });
  }
  if (metrics?.metrics_as_of) {
    fields.push({ label: 'Metrics as of', value: metrics.metrics_as_of });
  }
  return fields;
}

export type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
  SortOrder,
} from '@/domains/campaigns/list/campaigns_list_types';

export function CampaignsDirectory({
  items,
  total,
  limit,
  offset,
  statusTotals,
  statusTotalsLoading,
  customerOptions,
  customersLoading,
  customerNameById,
  metricsById,
  marginsById,
  draftCustomerId,
  draftStatus,
  draftPacing,
  draftOwnerUserId,
  draftCountry,
  draftBudgetMinUsd,
  draftBudgetMaxUsd,
  draftStatsFrom,
  draftStatsTo,
  ownerOptions,
  ownerEmailById,
  countryOptions,
  listFacetsFetching = false,
  listFacetsDegraded = false,
  listLastUpdatedAt = null,
  listScopeKey,
  statsQuery,
  exportFilterQuery,
  fetching,
  listRevalidating = false,
  error,
  metricsError,
  filterTotalsError,
  hasSnapshot,
  filtersActive,
  customerId,
  createCustomerId,
  createSectionOpen,
  onCreateSectionOpenChange,
  templates,
  templatesLoading,
  templatesError,
  draftTemplateId,
  draftCreateName,
  draftBudgetLimitMicro,
  creating,
  actionError,
  onDraftCustomerIdChange,
  onDraftStatusChange,
  onDraftPacingChange,
  onDraftOwnerUserIdChange,
  onDraftCountryChange,
  onDirectoryFiltersApply,
  onDraftBudgetMinUsdChange,
  onDraftBudgetMaxUsdChange,
  onStatsRangeChange,
  onRefreshList,
  onPageChange,
  onPageSizeChange,
  onDraftTemplateIdChange,
  onDraftCreateCustomerIdChange,
  onDraftCreateNameChange,
  onDraftBudgetLimitMicroChange,
  onLoadTemplates,
  onCreateCampaign,
}: CampaignsDirectoryProps) {
  const location = useLocation();
  const exportHubHref = useMemo(
    () =>
      buildExportHubHref({
        returnTo: `${location.pathname}${location.search}`,
      }),
    [location.pathname, location.search]
  );

  useEffect(() => {
    rememberExportHubReturnPath(`${location.pathname}${location.search}`);
  }, [location.pathname, location.search]);

  const workspace = useCampaignsDirectoryWorkspace({
    items,
    customerOptions,
    customerNameById,
    ownerEmailById,
    metricsById,
    marginsById,
    exportFilterQuery,
    listScopeKey,
    statsQuery,
    onRefreshList,
  });
  const [selectionGuardError, setSelectionGuardError] = useState<
    AdminValidationError | undefined
  >();

  useEffect(() => {
    setSelectionGuardError(undefined);
  }, [workspace.selectedIds]);

  const requireSelectedCampaigns = useCallback(
    (hint: string, action: () => void) => {
      if (workspace.selectedIds.size === 0) {
        const err = actionGuardError(hint);
        setSelectionGuardError(err);
        toastValidationError(err);
        return;
      }
      setSelectionGuardError(undefined);
      action();
    },
    [workspace.selectedIds.size]
  );

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const effectiveCreateCustomerId = createCustomerId.trim() || customerId || '';
  const createDisabled =
    creating || !effectiveCreateCustomerId || !draftTemplateId || templatesLoading;
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, (items ?? []).length);
  const rangeLabel = total === 0 ? '0 of 0' : `Showing ${rangeStart}-${rangeEnd} of ${total}`;
  const page = Math.floor(offset / limit) + 1;
  const pageCount = total === 0 ? 1 : Math.ceil(total / limit);

  const overlaysBusy =
    workspace.bulkBusy || workspace.exportBusy || createSectionOpen || workspace.wizardOpen;

  const handleCreateClick = useCallback(() => {
    if (overlaysBusy && !createSectionOpen) {
      return;
    }
    openCampaignCreateDialog(onCreateSectionOpenChange, workspace.setWizardOpen);
  }, [createSectionOpen, onCreateSectionOpenChange, overlaysBusy, workspace.setWizardOpen]);

  const handleWizardClick = useCallback(() => {
    if (overlaysBusy && !workspace.wizardOpen) {
      return;
    }
    openCampaignWizardSheet(onCreateSectionOpenChange, workspace.setWizardOpen);
  }, [onCreateSectionOpenChange, overlaysBusy, workspace.setWizardOpen, workspace.wizardOpen]);

  const handleCreateSectionOpenChange = useCallback(
    (open: boolean) => {
      setCampaignCreateDialogOpen(open, onCreateSectionOpenChange, workspace.setWizardOpen);
    },
    [onCreateSectionOpenChange, workspace.setWizardOpen]
  );

  const handleWizardOpenChange = useCallback(
    (open: boolean) => {
      setCampaignWizardSheetOpen(open, onCreateSectionOpenChange, workspace.setWizardOpen);
    },
    [onCreateSectionOpenChange, workspace.setWizardOpen]
  );

  const directoryMetricsError =
    metricsError && !isCampaignListAuxEndpointUnavailable(metricsError) ? metricsError : undefined;
  const directoryFilterTotalsError =
    filterTotalsError && !isCampaignListAuxEndpointUnavailable(filterTotalsError)
      ? filterTotalsError
      : undefined;

  const handlePauseSelected = useCallback(() => {
    runCampaignListBulkAction(workspace.bulkBusy, true, '', () =>
      requireSelectedCampaigns('Select at least one campaign', workspace.onPauseSelected)
    );
  }, [requireSelectedCampaigns, workspace.bulkBusy, workspace.onPauseSelected]);

  const handleResumeSelected = useCallback(() => {
    runCampaignListBulkAction(workspace.bulkBusy, true, '', () =>
      requireSelectedCampaigns('Select at least one campaign', workspace.onResumeSelected)
    );
  }, [requireSelectedCampaigns, workspace.bulkBusy, workspace.onResumeSelected]);

  const handleArchiveSelected = useCallback(() => {
    runCampaignListBulkAction(workspace.bulkBusy, true, '', () =>
      requireSelectedCampaigns('Select at least one campaign', () => workspace.setArchiveOpen(true))
    );
  }, [requireSelectedCampaigns, workspace.bulkBusy, workspace.setArchiveOpen]);

  const handleBulkEditSelected = useCallback(() => {
    runCampaignListBulkAction(workspace.bulkBusy, true, '', () =>
      requireSelectedCampaigns('Select at least one campaign', () =>
        workspace.setBulkPatchOpen(true)
      )
    );
  }, [requireSelectedCampaigns, workspace.bulkBusy, workspace.setBulkPatchOpen]);

  const handleClearSelection = useCallback(() => {
    workspace.setSelectedIds(new Set());
  }, [workspace.setSelectedIds]);

  useCampaignsCommandPaletteActions({
    selectedCount: workspace.selectedIds.size,
    onPauseSelected: handlePauseSelected,
    onResumeSelected: handleResumeSelected,
  });

  const selectedCampaignId = workspace.selectedCampaignId ?? null;
  const campaignById = useMemo(() => directoryRecordMap(items, (campaign) => campaign.id), [items]);
  const operateRows = useMemo(
    () =>
      directoryOperateRows(
        items,
        (campaign) => campaign.id,
        (campaign) => (
          <CampaignListNameWithSignals campaign={campaign} metrics={metricsById[campaign.id]} />
        )
      ),
    [items, metricsById]
  );
  const buildOverviewFields = useCallback(
    (campaign: Campaign) =>
      buildCampaignOverviewFields(campaign, customerNameById, metricsById[campaign.id]),
    [customerNameById, metricsById]
  );

  return (
    <>
      <DirectoryPageShell
        alerts={
          <>
            {selectionGuardError ? (
              <ValidationErrorBlock error={selectionGuardError} title="Select a campaign first" />
            ) : null}
            <DirectoryFetchError
              error={directoryMetricsError}
              fetchState={{ fetching, error: directoryMetricsError, hasSnapshot }}
              refreshTitle="Could not refresh campaign metrics"
              title="Could not load campaign metrics"
            />
            <DirectoryFetchError
              error={directoryFilterTotalsError}
              fetchState={{ fetching, error: directoryFilterTotalsError, hasSnapshot }}
              refreshTitle="Could not refresh filter totals"
              title="Could not load filter totals"
            />
          </>
        }
        blockingErrorTitle="Could not load campaigns"
        fillViewport={false}
        footerClassName="border-t-0 px-0 pt-2"
        headerClassName="border-b-0"
        mainClassName="min-w-0 w-full flex-none"
        workspaceClassName="pt-2"
        controlPanel={
          <div>
            {listFacetsDegraded ? (
              <StubBanner
                title="Owner and country filters limited"
                message={CAMPAIGN_LIST_FACETS_DEGRADED_MESSAGE}
              />
            ) : null}
            <CampaignsListToolbar
              countryOptions={countryOptions}
              customerOptions={customerOptions}
              draftStatsFrom={draftStatsFrom}
              draftStatsTo={draftStatsTo}
              draftBudgetMaxUsd={draftBudgetMaxUsd}
              draftBudgetMinUsd={draftBudgetMinUsd}
              draftCountry={draftCountry}
              draftCustomerId={draftCustomerId}
              draftStatus={draftStatus}
              draftOwnerUserId={draftOwnerUserId}
              draftPacing={draftPacing}
              fetching={fetching}
              filterFooter={
                workspace.selectedIds.size > 0 ? (
                  <CampaignsSelectionPanel
                    bulkBusy={workspace.bulkBusy}
                    exportBusy={workspace.exportBusy}
                    selectedCampaign={workspace.selectedCampaign}
                    selectedCount={workspace.selectedIds.size}
                    selectionGuardError={selectionGuardError}
                    variant="inline"
                    onArchive={handleArchiveSelected}
                    onBulkEdit={handleBulkEditSelected}
                    onClearSelection={handleClearSelection}
                    onClone={() => {
                      runCampaignListBulkAction(workspace.bulkBusy, true, '', () =>
                        requireSelectedCampaigns('Select a campaign first', () =>
                          workspace.setCloneOpen(true)
                        )
                      );
                    }}
                    onExportBundles={workspace.onExportBundles}
                    onExportCsv={workspace.onExportCsv}
                    onPause={handlePauseSelected}
                    onResume={handleResumeSelected}
                  />
                ) : null
              }
              listFacetsFetching={listFacetsFetching}
              listFacetsDegraded={listFacetsDegraded}
              listLastUpdatedAt={listLastUpdatedAt}
              listRevalidating={listRevalidating}
              overlaysBusy={overlaysBusy}
              ownerOptions={ownerOptions}
              statusTotals={statusTotals}
              statusTotalsLoading={statusTotalsLoading}
              onCreateClick={handleCreateClick}
              onDirectoryFiltersApply={onDirectoryFiltersApply}
              onStatsRangeChange={onStatsRangeChange}
              onDraftBudgetMaxUsdChange={onDraftBudgetMaxUsdChange}
              onDraftBudgetMinUsdChange={onDraftBudgetMinUsdChange}
              onDraftCountryChange={onDraftCountryChange}
              onDraftCustomerIdChange={onDraftCustomerIdChange}
              onDraftOwnerUserIdChange={onDraftOwnerUserIdChange}
              onDraftPacingChange={onDraftPacingChange}
              onDraftStatusChange={(status) => {
                workspace.setSelectedIds(new Set());
                onDraftStatusChange(status);
              }}
              onImportClick={() => workspace.setImportOpen(true)}
              onRefresh={onRefreshList}
              onWizardClick={handleWizardClick}
            />
          </div>
        }
        description={
          <>
            Create, edit, pause, and bulk-manage campaigns. BI report jobs run on{' '}
            <InAppLink className="text-primary hover:underline" to={exportHubHref}>
              Export hub
            </InAppLink>{' '}
            (Operations nav).
          </>
        }
        fetchState={{ fetching, error, hasSnapshot }}
        footer={
          <DirectoryPaginationFooter
            canGoNext={canGoNext}
            canGoPrev={canGoPrev}
            disabled={fetching}
            limit={limit}
            page={page}
            pageCount={pageCount}
            pageSizeId="campaigns-page-size"
            rangeLabel={rangeLabel}
            showPrevNext={false}
            onLimitChange={onPageSizeChange}
            onNext={() => onPageChange(offset + limit)}
            onPageChange={(nextPage) => onPageChange((nextPage - 1) * limit)}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
          />
        }
        skeletonColumns={2}
        title="Campaigns"
      >
        <DirectorySelectOverviewTable
          buildOverviewFields={buildOverviewFields}
          disabled={fetching}
          emptyMessage={
            filtersActive
              ? 'No campaigns match the current filters.'
              : 'No campaigns yet. Create one to start tracking spend and delivery.'
          }
          nameColumnLabel="Campaign"
          overviewFooter={(campaign) => (
            <>
              {campaign.id ? (
                <PrimaryActionButton asChild>
                  <Link to={`/campaigns/${campaign.id}/edit`}>Edit</Link>
                </PrimaryActionButton>
              ) : null}
              <SecondaryActionButton asChild>
                <InAppLink to={exportHubHref}>Export hub</InAppLink>
              </SecondaryActionButton>
            </>
          )}
          overviewTitle={(campaign) => campaign.name ?? campaign.id ?? ''}
          recordById={campaignById}
          revalidating={listRevalidating}
          rows={operateRows}
          selectedId={selectedCampaignId}
          onSelectedIdChange={(id) => workspace.setSelectedIds(id ? new Set([id]) : new Set())}
        />
      </DirectoryPageShell>

      <CampaignsDirectoryOverlays
        actionError={actionError}
        archiveOpen={workspace.archiveOpen}
        bulkBusy={workspace.bulkBusy}
        bulkCloneOpen={workspace.bulkCloneOpen}
        bulkPatchOpen={workspace.bulkPatchOpen}
        cloneOpen={workspace.cloneOpen}
        createDisabled={createDisabled}
        createSectionOpen={createSectionOpen}
        createCustomerId={createCustomerId}
        creating={creating}
        customerId={customerId}
        customerNameById={customerNameById}
        customerOptions={customerOptions}
        customersLoading={customersLoading}
        draftBudgetLimitMicro={draftBudgetLimitMicro}
        draftCreateName={draftCreateName}
        draftTemplateId={draftTemplateId}
        importOpen={workspace.importOpen}
        importPanelWorkspace={workspace.importPanelWorkspace}
        onArchiveConfirm={workspace.onArchiveSelected}
        onArchiveOpenChange={workspace.setArchiveOpen}
        onBulkCloneOpenChange={workspace.setBulkCloneOpen}
        onBulkPatchOpenChange={workspace.setBulkPatchOpen}
        onBulkPatched={() => {
          workspace.setBulkPatchOpen(false);
          workspace.setSelectedIds(new Set());
          workspace.refreshListAfterMutation();
        }}
        onCloneOpenChange={workspace.setCloneOpen}
        onBulkCloned={() => {
          workspace.setBulkCloneOpen(false);
          workspace.setSelectedIds(new Set());
          workspace.refreshListAfterMutation();
        }}
        onCloned={() => {
          workspace.setCloneOpen(false);
          workspace.refreshListAfterMutation();
        }}
        onCreateCampaign={onCreateCampaign}
        onCreateSectionOpenChange={handleCreateSectionOpenChange}
        onDraftBudgetLimitMicroChange={onDraftBudgetLimitMicroChange}
        onDraftCreateCustomerIdChange={onDraftCreateCustomerIdChange}
        onDraftCreateNameChange={onDraftCreateNameChange}
        onDraftTemplateIdChange={onDraftTemplateIdChange}
        onImportOpenChange={workspace.setImportOpen}
        onLoadTemplates={onLoadTemplates}
        onWizardOpenChange={handleWizardOpenChange}
        onWizardRefresh={workspace.refreshListAfterMutation}
        selectedCampaignId={workspace.selectedCampaignId}
        selectedCampaignIds={workspace.selectedIdsList}
        selectedCampaignName={workspace.selectedCampaign?.name}
        selectedCount={workspace.selectedIds.size}
        templates={templates}
        templatesError={templatesError}
        templatesLoading={templatesLoading}
        wizardOpen={workspace.wizardOpen}
        wizardPanelWorkspace={workspace.wizardPanelWorkspace}
      />
    </>
  );
}
