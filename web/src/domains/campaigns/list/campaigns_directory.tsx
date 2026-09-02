import { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { toast } from 'sonner';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignStatusTotals } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, SelfServeCampaignTemplate } from '@/api/types';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { PageLayout } from '@/shell/page_layout';
import { CampaignCloneDialog } from '@/domains/campaigns/editor/campaign_clone_dialog';
import { CampaignImportPanel } from '@/domains/campaigns/editor/campaign_import_panel';
import { CampaignOverviewSheet } from '@/domains/campaigns/list/campaign_overview_sheet';
import { CampaignWizardPanel } from '@/domains/campaigns/editor/campaign_wizard_panel';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import {
  archiveCampaigns,
  bulkPauseOrResumeCampaigns,
} from '@/domains/campaigns/list/campaign_list_bulk_actions';
import {
  exportCampaignBundles,
  exportVisibleRowsCsv,
} from '@/domains/campaigns/list/campaign_list_export';
import { computeCampaignListSummary } from '@/domains/campaigns/list/campaign_list_summary';
import type { CampaignsListFilterOption } from '@/domains/campaigns/list/campaigns_list_filter_select';
import {
  loadCampaignListColumnPrefs,
  mergeCampaignListColumnWidths,
  saveCampaignListColumnPrefs,
  setCampaignListColumnWidth,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  visibleCampaignListColumns,
} from '@/domains/campaigns/list/campaign_list_columns';
import {
  loadCampaignListRowDisplayPrefs,
  saveCampaignListRowDisplayPrefs,
  type CampaignListRowDisplayPrefs,
} from '@/domains/campaigns/list/campaign_list_row_display';
import { CampaignListResetWorkspaceDialog } from '@/domains/campaigns/list/campaign_list_reset_workspace_dialog';
import { resetCampaignListWorkspacePrefs } from '@/domains/campaigns/list/campaign_list_workspace_prefs';
import {
  computeCampaignListColumnWidths,
  defaultCampaignListColumnWidths,
} from '@/domains/campaigns/list/campaign_list_column_widths';
import { CampaignsListTable } from '@/domains/campaigns/list/campaigns_list_table';
import { CampaignsListToolbar } from '@/domains/campaigns/list/campaigns_list_toolbar';
import {
  type CampaignPacingFilter,
  type CampaignSortField,
  type CampaignStatusFilter,
  type SortOrder,
} from '@/domains/campaigns/list/campaigns_list_types';
import { listPageRange } from '@/lib/list_page_stats';
import { clampListLimit } from '@/lib/list_query';

export type {
  CampaignPacingFilter,
  CampaignSortField,
  CampaignStatusFilter,
  SortOrder,
} from '@/domains/campaigns/list/campaigns_list_types';

export type CampaignListColumnWidthProbe = {
  items: Campaign[];
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
};

export type CampaignsDirectoryProps = {
  items: Campaign[];
  total: number;
  limit: number;
  offset: number;
  statusTotals: CampaignStatusTotals | undefined;
  statusTotalsLoading: boolean;
  customerOptions: CustomerComboboxOption[];
  customersLoading: boolean;
  customerNameById: Record<string, string>;
  metricsById: Record<string, CampaignListMetrics>;
  marginsById: Record<string, CampaignMargin>;
  columnWidthProbe?: CampaignListColumnWidthProbe;
  appliedCustomerId: string;
  appliedStatus: CampaignStatusFilter;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  draftCustomerId: string;
  draftStatus: CampaignStatusFilter;
  draftPacing: CampaignPacingFilter;
  draftOwnerUserId: string;
  draftCountry: string;
  draftBudgetMinUsd: string;
  draftBudgetMaxUsd: string;
  draftStatsFrom: string;
  draftStatsTo: string;
  ownerOptions: CampaignsListFilterOption[];
  ownerEmailById: Record<string, string>;
  countryOptions: CampaignsListFilterOption[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  filtersActive: boolean;
  customerId: string | undefined;
  createSectionOpen: boolean;
  onCreateSectionOpenChange: (open: boolean) => void;
  templates: SelfServeCampaignTemplate[];
  templatesLoading: boolean;
  templatesError: Error | undefined;
  draftTemplateId: string;
  draftCreateName: string;
  draftBudgetLimitMicro: string;
  creating: boolean;
  actionError: Error | undefined;
  onDraftCustomerIdChange: (customerId: string) => void;
  onDraftStatusChange: (status: CampaignStatusFilter) => void;
  onDraftPacingChange: (pacing: CampaignPacingFilter) => void;
  onDraftOwnerUserIdChange: (ownerUserId: string) => void;
  onDraftCountryChange: (country: string) => void;
  onDraftBudgetMinUsdChange: (value: string) => void;
  onDraftBudgetMaxUsdChange: (value: string) => void;
  onBudgetFiltersApply: () => void;
  onStatsRangeChange: (from: string, to: string) => void;
  onRefreshList: () => void;
  onColumnSort: (field: CampaignSortField) => void;
  onPageChange: (nextOffset: number) => void;
  onPageSizeChange: (size: number) => void;
  onDraftTemplateIdChange: (templateId: string) => void;
  onDraftCreateNameChange: (name: string) => void;
  onDraftBudgetLimitMicroChange: (value: string) => void;
  onLoadTemplates: () => void;
  onCreateCampaign: () => void;
};

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
  const navigate = useNavigate();
  const [importOpen, setImportOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [cloneOpen, setCloneOpen] = useState(false);
  const [archiveOpen, setArchiveOpen] = useState(false);
  const [resetWorkspaceOpen, setResetWorkspaceOpen] = useState(false);
  const [bulkBusy, setBulkBusy] = useState(false);
  const [exportBusy, setExportBusy] = useState(false);
  const [overviewCampaign, setOverviewCampaign] = useState<CampaignWithMoneyDisplay | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [columnPrefs, setColumnPrefs] = useState<CampaignListColumnPrefs>(() =>
    loadCampaignListColumnPrefs(),
  );
  const [rowDisplayPrefs, setRowDisplayPrefs] = useState<CampaignListRowDisplayPrefs>(() =>
    loadCampaignListRowDisplayPrefs(),
  );
  const [pageSizeDraft, setPageSizeDraft] = useState(String(limit));

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const createDisabled =
    creating || !customerId || !draftTemplateId || templatesLoading;
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, items.length);
  const rangeLabel = total === 0 ? '0 of 0' : `${rangeStart} - ${rangeEnd} of ${total}`;

  const handleColumnPrefsApply = useCallback((prefs: CampaignListColumnPrefs) => {
    setColumnPrefs(prefs);
    saveCampaignListColumnPrefs(prefs);
  }, []);

  const handleRowDisplayPrefsChange = useCallback((prefs: CampaignListRowDisplayPrefs) => {
    setRowDisplayPrefs(prefs);
    saveCampaignListRowDisplayPrefs(prefs);
  }, []);

  const handleResetWorkspaceConfirm = useCallback(() => {
    const prefs = resetCampaignListWorkspacePrefs();
    setColumnPrefs(prefs.columnPrefs);
    setRowDisplayPrefs(prefs.rowDisplayPrefs);
    setResetWorkspaceOpen(false);
    toast.success('Campaign list view reset');
  }, []);

  const visibleColumns = useMemo(
    () => visibleCampaignListColumns(columnPrefs),
    [columnPrefs],
  );

  const computedColumnWidths = useMemo((): Record<CampaignListColumnId, number> => {
    if (!columnWidthProbe?.items.length) {
      return defaultCampaignListColumnWidths(visibleColumns);
    }
    return computeCampaignListColumnWidths({
      columns: visibleColumns,
      items: columnWidthProbe.items,
      metricsById: columnWidthProbe.metricsById,
      marginsById: columnWidthProbe.marginsById,
      customerNameById,
    });
  }, [columnWidthProbe, customerNameById, visibleColumns]);

  const columnWidths = useMemo(
    () => mergeCampaignListColumnWidths(computedColumnWidths, columnPrefs.widthPx, visibleColumns),
    [columnPrefs.widthPx, computedColumnWidths, visibleColumns],
  );

  const handleColumnWidthCommit = useCallback((columnId: CampaignListColumnId, widthPx: number) => {
    setColumnPrefs((current) => {
      const next = setCampaignListColumnWidth(current, columnId, widthPx);
      saveCampaignListColumnPrefs(next);
      return next;
    });
  }, []);

  const handlePageSizeCommit = useCallback(
    (raw: string) => {
      const next = clampListLimit(Number.parseInt(raw, 10));
      setPageSizeDraft(String(next));
      if (next !== limit) {
        onPageSizeChange(next);
      }
    },
    [limit, onPageSizeChange],
  );

  const selectedCampaignId = useMemo(() => {
    if (selectedIds.size !== 1) {
      return undefined;
    }
    return [...selectedIds][0];
  }, [selectedIds]);

  const selectedCampaign = useMemo(
    () => items.find((item) => item.id === selectedCampaignId),
    [items, selectedCampaignId],
  );

  const summary = useMemo(
    () => computeCampaignListSummary(items, selectedIds, metricsById, marginsById),
    [items, marginsById, metricsById, selectedIds],
  );

  const selectedIdsList = useMemo(() => [...selectedIds], [selectedIds]);

  const runBulkAction = useCallback(
    async (label: string, action: () => Promise<{ succeeded: string[]; failed: { id: string; error: string }[] }>) => {
      if (selectedIdsList.length === 0) {
        return;
      }
      setBulkBusy(true);
      try {
        const result = await action();
        if (result.succeeded.length > 0) {
          toast.success(`${label}: ${result.succeeded.length} campaign(s)`);
        }
        if (result.failed.length > 0) {
          toast.error(`${label} failed for ${result.failed.length} campaign(s)`);
        }
        setSelectedIds(new Set());
        onRefreshList();
      } catch (err) {
        toast.error(err instanceof Error ? err.message : String(err));
      } finally {
        setBulkBusy(false);
        setArchiveOpen(false);
      }
    },
    [onRefreshList, selectedIdsList],
  );

  const onPauseSelected = useCallback(() => {
    void runBulkAction('Paused', () => bulkPauseOrResumeCampaigns('pause', selectedIdsList));
  }, [runBulkAction, selectedIdsList]);

  const onResumeSelected = useCallback(() => {
    void runBulkAction('Resumed', () => bulkPauseOrResumeCampaigns('resume', selectedIdsList));
  }, [runBulkAction, selectedIdsList]);

  const onArchiveSelected = useCallback(() => {
    void runBulkAction('Archived', () => archiveCampaigns(selectedIdsList));
  }, [runBulkAction, selectedIdsList]);

  const onExportBundles = useCallback(() => {
    const ids = selectedIdsList.length > 0 ? selectedIdsList : items.map((item) => item.id);
    if (ids.length === 0) {
      return;
    }
    setExportBusy(true);
    void exportCampaignBundles(ids)
      .then(() => toast.success('Campaign export downloaded'))
      .catch((err: unknown) => {
        toast.error(err instanceof Error ? err.message : String(err));
      })
      .finally(() => setExportBusy(false));
  }, [items, selectedIdsList]);

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
          bulkBusy={bulkBusy || exportBusy}
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
          summary={summary}
          selectedCount={selectedIds.size}
          onArchiveClick={() => {
            if (selectedIds.size === 0) {
              toast.error('Select at least one campaign');
              return;
            }
            setArchiveOpen(true);
          }}
          onBudgetFiltersApply={onBudgetFiltersApply}
          onCloneClick={() => {
            if (selectedIds.size !== 1) {
              toast.error('Select exactly one campaign to clone');
              return;
            }
            setCloneOpen(true);
          }}
          columnPrefs={columnPrefs}
          onColumnPrefsChange={handleColumnPrefsApply}
          rowDisplayPrefs={rowDisplayPrefs}
          onRowDisplayPrefsChange={handleRowDisplayPrefsChange}
          onCreateClick={() => onCreateSectionOpenChange(true)}
          onStatsRangeChange={onStatsRangeChange}
          onDraftBudgetMaxUsdChange={onDraftBudgetMaxUsdChange}
          onDraftBudgetMinUsdChange={onDraftBudgetMinUsdChange}
          onDraftCountryChange={onDraftCountryChange}
          onDraftCustomerIdChange={onDraftCustomerIdChange}
          onDraftOwnerUserIdChange={onDraftOwnerUserIdChange}
          onDraftPacingChange={onDraftPacingChange}
          onDraftStatusChange={onDraftStatusChange}
          onImportClick={() => setImportOpen(true)}
          onPauseClick={() => {
            if (selectedIds.size === 0) {
              toast.error('Select at least one campaign');
              return;
            }
            onPauseSelected();
          }}
          onRefresh={onRefreshList}
          onResetWorkspaceClick={() => setResetWorkspaceOpen(true)}
          onReportClick={() => {
            if (selectedIds.size !== 1) {
              toast.error('Select exactly one campaign for report');
              return;
            }
            if (selectedCampaignId) {
              navigate(`/dashboards/campaign/${selectedCampaignId}`);
            }
          }}
          onResumeClick={() => {
            if (selectedIds.size === 0) {
              toast.error('Select at least one campaign');
              return;
            }
            onResumeSelected();
          }}
          onWizardClick={() => setWizardOpen(true)}
        />
      }
      footer={
        <div className="admin-footer-bar">
          <div aria-label="Pagination" className="admin-footer-pagination">
            <Button
              aria-label="Previous page"
              disabled={fetching || !canGoPrev}
              type="button"
              variant="secondary"
              onClick={() => onPageChange(Math.max(0, offset - limit))}
            >
              Prev
            </Button>
            <Button
              aria-label="Next page"
              disabled={fetching || !canGoNext}
              type="button"
              variant="secondary"
              onClick={() => onPageChange(offset + limit)}
            >
              Next
            </Button>
            <label className="admin-label">
              Per page
              <input
                className="admin-select"
                disabled={fetching}
                inputMode="numeric"
                value={pageSizeDraft}
                onBlur={() => handlePageSizeCommit(pageSizeDraft)}
                onChange={(event) => setPageSizeDraft(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter') {
                    handlePageSizeCommit(pageSizeDraft);
                  }
                }}
              />
            </label>
            <span className="admin-muted tabular-nums">{rangeLabel}</span>
          </div>
          <div aria-label="Export" className="admin-footer-exports">
            <Button
              disabled={fetching || items.length === 0 || exportBusy}
              title="Download visible table rows as CSV"
              type="button"
              variant="secondary"
              onClick={() => exportVisibleRowsCsv(items, customerNameById)}
            >
              Export CSV
            </Button>
            <Button
              disabled={fetching || items.length === 0 || exportBusy}
              title="Download campaign bundle JSON for selected rows, or all rows on this page"
              type="button"
              variant="secondary"
              onClick={onExportBundles}
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
        columnPrefs={columnPrefs}
        columnWidths={columnWidths}
        highlightActiveRows={rowDisplayPrefs.highlightActiveRows}
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
        selectedIds={selectedIds}
        onColumnPrefsChange={handleColumnPrefsApply}
        onColumnSort={onColumnSort}
        onColumnWidthCommit={handleColumnWidthCommit}
        onCampaignOverview={(campaign) => setOverviewCampaign(campaign as CampaignWithMoneyDisplay)}
        onSelectedIdsChange={setSelectedIds}
      />
    </PageLayout>

      {actionError ? <ErrorBlock title="Action failed" message={actionError.message} /> : null}

      <Dialog open={createSectionOpen} onOpenChange={onCreateSectionOpenChange}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>Create campaign</DialogTitle>
            <DialogDescription>
              {customerId ? (
                <>
                  Customer{' '}
                  <span className="font-mono text-xs text-foreground">{customerId}</span>
                </>
              ) : (
                'Select a customer group filter to create a campaign.'
              )}
            </DialogDescription>
          </DialogHeader>

          <form
            className="grid gap-4"
            onSubmit={(event) => {
              event.preventDefault();
              onCreateCampaign();
            }}
          >
            <div className="grid gap-2">
              <Label htmlFor="campaigns-template">Template</Label>
              <Select
                disabled={!customerId || templates.length === 0}
                value={draftTemplateId}
                onValueChange={onDraftTemplateIdChange}
              >
                <SelectTrigger className="w-full" id="campaigns-template">
                  <SelectValue placeholder={templatesLoading ? 'Loading\u2026' : 'Select template\u2026'} />
                </SelectTrigger>
                <SelectContent plain>
                  {templates.map((template) => (
                    <SelectItem key={template.id} plain value={template.id}>
                      {template.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="campaigns-create-name">Name</Label>
              <Input
                id="campaigns-create-name"
                disabled={!customerId}
                placeholder="Optional display name\u2026"
                value={draftCreateName}
                onChange={(event) => onDraftCreateNameChange(event.target.value)}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="campaigns-budget-micro">Budget (micro)</Label>
              <Input
                id="campaigns-budget-micro"
                disabled={!customerId}
                inputMode="numeric"
                placeholder="Optional override\u2026"
                value={draftBudgetLimitMicro}
                onChange={(event) => onDraftBudgetLimitMicroChange(event.target.value)}
              />
            </div>

            {templatesError ? (
              <ErrorBlock title="Could not load templates" message={templatesError.message} />
            ) : null}
            {customerId && !templatesLoading && templates.length === 0 && !templatesError ? (
              <p className="text-sm text-muted-foreground">No templates for this customer.</p>
            ) : null}

            <DialogFooter className="gap-2 sm:gap-0">
              <SecondaryActionButton
                disabled={!customerId}
                loading={templatesLoading}
                onClick={onLoadTemplates}
                type="button"
              >
                Reload templates
              </SecondaryActionButton>
              <PrimaryActionButton disabled={createDisabled} loading={creating} type="submit">
                Create
              </PrimaryActionButton>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <CampaignCloneDialog
        campaignId={selectedCampaignId}
        campaignName={selectedCampaign?.name}
        open={cloneOpen}
        onCloned={() => {
          setCloneOpen(false);
          onRefreshList();
        }}
        onOpenChange={setCloneOpen}
      />

      <Dialog open={archiveOpen} onOpenChange={setArchiveOpen}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Archive campaigns</DialogTitle>
            <DialogDescription>
              Archive {selectedIds.size} selected campaign(s)? They can be filtered under Archived
              status.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <SecondaryActionButton type="button" onClick={() => setArchiveOpen(false)}>
              Cancel
            </SecondaryActionButton>
            <PrimaryActionButton loading={bulkBusy} type="button" onClick={onArchiveSelected}>
              Archive
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <CampaignListResetWorkspaceDialog
        open={resetWorkspaceOpen}
        onConfirm={handleResetWorkspaceConfirm}
        onOpenChange={setResetWorkspaceOpen}
      />

      <CampaignOverviewSheet
        campaign={overviewCampaign}
        customerName={
          overviewCampaign
            ? customerNameById[overviewCampaign.customer_id] ?? overviewCampaign.customer_id
            : ''
        }
        onOpenChange={(open) => {
          if (!open) {
            setOverviewCampaign(null);
          }
        }}
        open={overviewCampaign != null}
      />

      <Sheet onOpenChange={setImportOpen} open={importOpen}>
        <SheetContent className="flex h-full w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-2xl">
          <SheetHeader className="shrink-0 border-b border-[var(--admin-border)] px-6 py-4 text-left">
            <SheetTitle>Import campaign</SheetTitle>
            <SheetDescription>Validate, migrate, or import a campaign bundle.</SheetDescription>
          </SheetHeader>
          <div className="ui-scrollbar min-h-0 flex-1 overflow-y-auto px-6 py-4 pb-8">
            <CampaignImportPanel />
          </div>
        </SheetContent>
      </Sheet>

      <Sheet onOpenChange={setWizardOpen} open={wizardOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Campaign wizard</SheetTitle>
            <SheetDescription>Guided setup for a new campaign.</SheetDescription>
          </SheetHeader>
          <div className="mt-6">
            <CampaignWizardPanel
              customerOptions={customerOptions}
              onCampaignCreated={() => onRefreshList()}
            />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}
