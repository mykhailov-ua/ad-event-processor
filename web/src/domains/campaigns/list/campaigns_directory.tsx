import { toast } from 'sonner';

import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { exportVisibleRowsCsv } from '@/domains/campaigns/list/campaign_list_export';
import { CampaignsListTable } from '@/domains/campaigns/list/campaigns_list_table';
import { CampaignsListToolbar } from '@/domains/campaigns/list/campaigns_list_toolbar';
import { CampaignsDirectoryOverlays } from '@/domains/campaigns/list/campaigns_directory_overlays';
import type { CampaignsDirectoryProps } from '@/domains/campaigns/list/campaigns_directory_types';
import { useCampaignsDirectoryWorkspace } from '@/domains/campaigns/list/use_campaigns_directory_workspace';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { PageLayout } from '@/shell/page_layout';
import { DirectoryPaginationFooter } from '@/shell/directory_pagination_footer';
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
  fetching,
  error,
  hasSnapshot,
  filtersActive,
  customerId,
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
  onDraftCreateNameChange,
  onDraftBudgetLimitMicroChange,
  onLoadTemplates,
  onCreateCampaign,
}: CampaignsDirectoryProps) {
  const workspace = useCampaignsDirectoryWorkspace({
    items,
    customerNameById,
    metricsById,
    marginsById,
    columnWidthProbe,
    onRefreshList,
  });

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const createDisabled =
    creating || !customerId || !draftTemplateId || templatesLoading;
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, items.length);
  const rangeLabel = total === 0 ? '0 of 0' : `${rangeStart} - ${rangeEnd} of ${total}`;

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
            columnPrefs={workspace.columnPrefs}
            onColumnPrefsChange={workspace.handleColumnPrefsApply}
            rowDisplayPrefs={workspace.rowDisplayPrefs}
            onRowDisplayPrefsChange={workspace.handleRowDisplayPrefsChange}
            onCreateClick={() => onCreateSectionOpenChange(true)}
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
            onResetWorkspaceClick={() => workspace.setResetWorkspaceOpen(true)}
            onReportClick={workspace.onReportClick}
            onResumeClick={() => {
              if (workspace.selectedIds.size === 0) {
                toast.error('Select at least one campaign');
                return;
              }
              workspace.onResumeSelected();
            }}
            onWizardClick={() => workspace.setWizardOpen(true)}
          />
        }
        footer={
          <div className="flex flex-wrap items-center gap-2">
            <DirectoryPaginationFooter
              canGoNext={canGoNext}
              canGoPrev={canGoPrev}
              disabled={fetching}
              limit={limit}
              pageSizeId="campaigns-page-size"
              prevLabel="Prev"
              rangeLabel={rangeLabel}
              onLimitChange={onPageSizeChange}
              onNext={() => onPageChange(offset + limit)}
              onPrev={() => onPageChange(Math.max(0, offset - limit))}
            />
            <div aria-label="Export" className="flex flex-wrap items-center gap-2">
              <Button
                disabled={fetching || items.length === 0 || workspace.exportBusy}
                title="Download visible table rows as CSV"
                type="button"
                variant="secondary"
                onClick={() => exportVisibleRowsCsv(items, customerNameById)}
              >
                Export CSV
              </Button>
              <Button
                disabled={fetching || items.length === 0 || workspace.exportBusy}
                title="Download campaign bundle JSON for selected rows, or all rows on this page"
                type="button"
                variant="secondary"
                onClick={workspace.onExportBundles}
              >
                Export JSON
              </Button>
            </div>
          </div>
        }
        title="Campaigns"
      >
        <CampaignsListTable
          appliedOrder={appliedOrder}
          appliedSort={appliedSort}
          columnPrefs={workspace.columnPrefs}
          columnWidths={workspace.columnWidths}
          highlightActiveRows={workspace.rowDisplayPrefs.highlightActiveRows}
          customerNameById={customerNameById}
          ownerEmailById={ownerEmailById}
          emptyMessage={
            filtersActive
              ? 'No campaigns match the current filters.'
              : 'No campaigns yet. Create one to start tracking spend and delivery.'
          }
          fetching={fetching}
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
        />
      </PageLayout>

      <CampaignsDirectoryOverlays
        actionError={actionError}
        archiveOpen={workspace.archiveOpen}
        bulkBusy={workspace.bulkBusy}
        cloneOpen={workspace.cloneOpen}
        createDisabled={createDisabled}
        createSectionOpen={createSectionOpen}
        creating={creating}
        customerId={customerId}
        customerNameById={customerNameById}
        customerOptions={customerOptions}
        draftBudgetLimitMicro={draftBudgetLimitMicro}
        draftCreateName={draftCreateName}
        draftTemplateId={draftTemplateId}
        importOpen={workspace.importOpen}
        onArchiveConfirm={workspace.onArchiveSelected}
        onArchiveOpenChange={workspace.setArchiveOpen}
        onCloneOpenChange={workspace.setCloneOpen}
        onCloned={() => {
          workspace.setCloneOpen(false);
          onRefreshList();
        }}
        onCreateCampaign={onCreateCampaign}
        onCreateSectionOpenChange={onCreateSectionOpenChange}
        onDraftBudgetLimitMicroChange={onDraftBudgetLimitMicroChange}
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
        onWizardOpenChange={workspace.setWizardOpen}
        onWizardRefresh={onRefreshList}
        overviewCampaign={workspace.overviewCampaign}
        resetWorkspaceOpen={workspace.resetWorkspaceOpen}
        selectedCampaignId={workspace.selectedCampaignId}
        selectedCampaignName={workspace.selectedCampaign?.name}
        selectedCount={workspace.selectedIds.size}
        templates={templates}
        templatesError={templatesError}
        templatesLoading={templatesLoading}
        wizardOpen={workspace.wizardOpen}
      />
    </>
  );
}
