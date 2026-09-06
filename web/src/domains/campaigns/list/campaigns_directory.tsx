import { useCallback } from 'react';
import { toast } from 'sonner';

import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { CampaignListTableCardTools } from '@/domains/campaigns/list/campaign_list_table_card_tools';
import { CampaignsListTable } from '@/domains/campaigns/list/campaigns_list_table';
import { CampaignsListToolbar } from '@/domains/campaigns/list/campaigns_list_toolbar';
import { CampaignsDirectoryOverlays } from '@/domains/campaigns/list/campaigns_directory_overlays';
import type { CampaignsDirectoryProps } from '@/domains/campaigns/list/campaigns_directory_types';
import { campaignListTableCardClass } from '@/domains/campaigns/list/campaign_list_classes';
import { useCampaignsDirectoryWorkspace } from '@/domains/campaigns/list/use_campaigns_directory_workspace';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { PageLayout } from '@/shell/page_layout';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
import { StubBanner } from '@/shell/stub_banner';
import { CAMPAIGN_LIST_FACETS_DEGRADED_MESSAGE } from '@/domains/campaigns/list/campaign_list_facets_source';
import {
  openCampaignCreateDialog,
  openCampaignWizardSheet,
  setCampaignCreateDialogOpen,
  setCampaignWizardSheetOpen,
} from '@/domains/campaigns/list/campaign_list_create_overlay';
import { Button } from '@/components/ui/button';
import { listPageRange } from '@/lib/list_page_stats';

export type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
  SortOrder,
  CampaignListColumnWidthProbe,
} from '@/domains/campaigns/list/campaigns_directory_types';

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
  columnWidthProbe,
  appliedSort,
  appliedOrder,
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
  filterTotals,
  filterTotalsCapped = false,
  filteredTotal = 0,
  metricsStale = false,
  listScopeKey,
  statsQuery,
  exportFilterQuery,
  fetching,
  error,
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
  onDraftBudgetMinUsdChange,
  onDraftBudgetMaxUsdChange,
  onBudgetFiltersApply,
  onStatsRangeChange,
  onRefreshList,
  onColumnSort,
  onPageChange,
  onPageSizeChange,
  onDraftTemplateIdChange,
  onDraftCreateCustomerIdChange,
  onDraftCreateNameChange,
  onDraftBudgetLimitMicroChange,
  onLoadTemplates,
  onCreateCampaign,
}: CampaignsDirectoryProps) {
  const workspace = useCampaignsDirectoryWorkspace({
    items,
    customerOptions,
    customerNameById,
    ownerEmailById,
    metricsById,
    marginsById,
    columnWidthProbe,
    filterTotals,
    exportFilterQuery,
    listScopeKey,
    statsQuery,
    onRefreshList,
  });

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const effectiveCreateCustomerId = createCustomerId.trim() || customerId || '';
  const createDisabled =
    creating || !effectiveCreateCustomerId || !draftTemplateId || templatesLoading;
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, (items ?? []).length);
  const rangeLabel = total === 0 ? '0 of 0' : `Showing ${rangeStart}-${rangeEnd} of ${total}`;
  const page = Math.floor(offset / limit) + 1;
  const pageCount = total === 0 ? 1 : Math.ceil(total / limit);

  const handleCreateClick = useCallback(() => {
    openCampaignCreateDialog(onCreateSectionOpenChange, workspace.setWizardOpen);
  }, [onCreateSectionOpenChange, workspace.setWizardOpen]);

  const handleWizardClick = useCallback(() => {
    openCampaignWizardSheet(onCreateSectionOpenChange, workspace.setWizardOpen);
  }, [onCreateSectionOpenChange, workspace.setWizardOpen]);

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

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={8} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load campaigns" message={error.message} />;
  }

  return (
    <>
      <PageLayout
        controlPanel={
          <div className="grid gap-3">
            {listFacetsDegraded ? (
              <StubBanner
                title="Owner and country filters limited"
                message={CAMPAIGN_LIST_FACETS_DEGRADED_MESSAGE}
              />
            ) : null}
            <CampaignsListToolbar
              bulkBusy={workspace.bulkBusy || workspace.exportBusy}
              countryOptions={countryOptions}
              customerOptions={customerOptions}
              draftStatsFrom={draftStatsFrom}
              draftStatsTo={draftStatsTo}
              draftBudgetMaxUsd={draftBudgetMaxUsd}
              draftBudgetMinUsd={draftBudgetMinUsd}
              draftCountry={draftCountry}
              draftCustomerId={draftCustomerId}
              appliedStatus={draftStatus}
              draftOwnerUserId={draftOwnerUserId}
              draftPacing={draftPacing}
              fetching={fetching}
              filterTotalsCapped={filterTotalsCapped}
              filteredTotal={filteredTotal}
              listFacetsFetching={listFacetsFetching}
              listFacetsDegraded={listFacetsDegraded}
              metricsStale={metricsStale}
              ownerOptions={ownerOptions}
              statusTotals={statusTotals}
              statusTotalsLoading={statusTotalsLoading}
              summary={workspace.summary}
              selectedCount={workspace.selectedIds.size}
              onArchiveClick={() => {
                if (workspace.selectedIds.size === 0) {
                  toast.error('Select at least one campaign');
                  return;
                }
                workspace.setArchiveOpen(true);
              }}
              onBudgetFiltersApply={onBudgetFiltersApply}
              onCloneClick={() => {
                if (workspace.selectedIds.size !== 1) {
                  toast.error('Select exactly one campaign to clone');
                  return;
                }
                workspace.setCloneOpen(true);
              }}
              onCreateClick={handleCreateClick}
              onStatsRangeChange={onStatsRangeChange}
              onDraftBudgetMaxUsdChange={onDraftBudgetMaxUsdChange}
              onDraftBudgetMinUsdChange={onDraftBudgetMinUsdChange}
              onDraftCountryChange={onDraftCountryChange}
              onDraftCustomerIdChange={onDraftCustomerIdChange}
              onDraftOwnerUserIdChange={onDraftOwnerUserIdChange}
              onDraftPacingChange={onDraftPacingChange}
              onDraftStatusChange={onDraftStatusChange}
              onImportClick={() => workspace.setImportOpen(true)}
              onPauseClick={() => {
                if (workspace.selectedIds.size === 0) {
                  toast.error('Select at least one campaign');
                  return;
                }
                workspace.onPauseSelected();
              }}
              onRefresh={onRefreshList}
              onReportClick={workspace.onReportClick}
              onResumeClick={() => {
                if (workspace.selectedIds.size === 0) {
                  toast.error('Select at least one campaign');
                  return;
                }
                workspace.onResumeSelected();
              }}
              onWizardClick={handleWizardClick}
              canGoNext={canGoNext}
              canGoPrev={canGoPrev}
              paginationDisabled={fetching}
              onPageNext={() => onPageChange(offset + limit)}
              onPagePrev={() => onPageChange(Math.max(0, offset - limit))}
            />
            <CampaignListTableCardTools
              columnPrefs={workspace.columnPrefs}
              disabled={fetching}
              onColumnPrefsChange={workspace.handleColumnPrefsApply}
              onResetWorkspaceClick={() => workspace.setResetWorkspaceOpen(true)}
            />
          </div>
        }
        footer={
          <div className="flex flex-wrap items-center gap-3">
            <DirectoryPaginationFooter
              canGoNext={canGoNext}
              canGoPrev={canGoPrev}
              className="gap-2"
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
            <div aria-label="Export" className="flex flex-wrap items-center gap-2">
              <Button
                disabled={fetching || total === 0 || workspace.exportBusy}
                title="Download CSV for selected campaigns, or all campaigns matching the current filters"
                type="button"
                variant="outline"
                onClick={workspace.onExportCsv}
              >
                Export CSV
              </Button>
              <Button
                disabled={fetching || total === 0 || workspace.exportBusy}
                title="Download JSON bundles for selected campaigns, or all campaigns matching the current filters"
                type="button"
                variant="outline"
                onClick={workspace.onExportBundles}
              >
                Export JSON
              </Button>
            </div>
          </div>
        }
      >
        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          <div className={campaignListTableCardClass}>
            <CampaignsListTable
              appliedOrder={appliedOrder}
              appliedSort={appliedSort}
              columnPrefs={workspace.columnPrefs}
              columnWidths={workspace.columnWidths}
              customerNameById={customerNameById}
              ownerEmailById={ownerEmailById}
              emptyMessage={
                filtersActive
                  ? 'No campaigns match the current filters.'
                  : 'No campaigns yet. Create one to start tracking spend and delivery.'
              }
              fetching={fetching}
              filterTotals={filterTotals}
              items={items}
              marginsById={marginsById}
              metricsById={metricsById}
              selectedIds={workspace.selectedIds}
              onColumnPrefsChange={workspace.handleColumnPrefsApply}
              onColumnSort={onColumnSort}
              onColumnWidthCommit={workspace.handleColumnWidthCommit}
              onCampaignOverview={(campaign) =>
                workspace.setOverviewCampaign(campaign as CampaignWithMoneyDisplay)
              }
              onSelectedIdsChange={workspace.setSelectedIds}
              statsCacheRevision={listScopeKey}
              statsQuery={statsQuery}
            />
          </div>
        </div>
      </PageLayout>

      <CampaignsDirectoryOverlays
        actionError={actionError}
        archiveOpen={workspace.archiveOpen}
        bulkBusy={workspace.bulkBusy}
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
        onCloneOpenChange={workspace.setCloneOpen}
        onCloned={() => {
          workspace.setCloneOpen(false);
          onRefreshList();
        }}
        onCreateCampaign={onCreateCampaign}
        onCreateSectionOpenChange={handleCreateSectionOpenChange}
        onDraftBudgetLimitMicroChange={onDraftBudgetLimitMicroChange}
        onDraftCreateCustomerIdChange={onDraftCreateCustomerIdChange}
        onDraftCreateNameChange={onDraftCreateNameChange}
        onDraftTemplateIdChange={onDraftTemplateIdChange}
        onImportOpenChange={workspace.setImportOpen}
        onLoadTemplates={onLoadTemplates}
        onOverviewOpenChange={(open) => {
          if (!open) {
            workspace.setOverviewCampaign(null);
          }
        }}
        onResetWorkspaceConfirm={workspace.handleResetWorkspaceConfirm}
        onResetWorkspaceOpenChange={workspace.setResetWorkspaceOpen}
        onWizardOpenChange={handleWizardOpenChange}
        onWizardRefresh={onRefreshList}
        overviewCampaign={workspace.overviewCampaign}
        marginsById={marginsById}
        metricsById={metricsById}
        listScopeKey={listScopeKey}
        resetWorkspaceOpen={workspace.resetWorkspaceOpen}
        selectedCampaignId={workspace.selectedCampaignId}
        selectedCampaignName={workspace.selectedCampaign?.name}
        selectedCount={workspace.selectedIds.size}
        statsQuery={statsQuery}
        templates={templates}
        templatesError={templatesError}
        templatesLoading={templatesLoading}
        wizardOpen={workspace.wizardOpen}
        wizardPanelWorkspace={workspace.wizardPanelWorkspace}
      />
    </>
  );
}
