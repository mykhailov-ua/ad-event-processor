import { useCallback, useMemo, useState } from 'react';
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';

import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { CampaignStatusTotals } from '@/api/campaigns_api';
import type { Campaign, CampaignMargin, SelfServeCampaignTemplate } from '@/api/types';
import type { CustomerComboboxOption } from '@/components/system/customer_combobox';
import { PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
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
import { CampaignImportPanel } from '@/domains/campaigns/campaign_import_panel';
import { CampaignOverviewSheet } from '@/domains/campaigns/campaign_overview_sheet';
import { CampaignWizardPanel } from '@/domains/campaigns/campaign_wizard_panel';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/campaign_metrics_shared';
import {
  loadCampaignListColumnPrefs,
  mergeCampaignListColumnWidths,
  saveCampaignListColumnPrefs,
  setCampaignListColumnWidth,
  type CampaignListColumnId,
  type CampaignListColumnPrefs,
  visibleCampaignListColumns,
} from '@/domains/campaigns/campaign_list_columns';
import {
  computeCampaignListColumnWidths,
  defaultCampaignListColumnWidths,
} from '@/domains/campaigns/campaign_list_column_widths';
import { CampaignsListTable } from '@/domains/campaigns/campaigns_list_table';
import { CampaignsListToolbar } from '@/domains/campaigns/campaigns_list_toolbar';
import {
  type CampaignSortField,
  type CampaignStatusFilter,
  type SortOrder,
} from '@/domains/campaigns/campaigns_list_types';
import { listPageRange } from '@/lib/list_page_stats';
import { clampListLimit } from '@/lib/list_query';
import { useTrackerHeaderSearchRegistration } from '@/lib/tracker_header_context';

export type { CampaignSortField, CampaignStatusFilter, SortOrder } from '@/domains/campaigns/campaigns_list_types';

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
  columnWidthProbe: CampaignListColumnWidthProbe | undefined;
  appliedCustomerId: string;
  appliedStatus: CampaignStatusFilter;
  appliedSort: CampaignSortField;
  appliedOrder: SortOrder;
  draftCustomerId: string;
  draftStatus: CampaignStatusFilter;
  draftQ: string;
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
  onDraftQChange: (q: string) => void;
  onSearchApply: () => void;
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

function exportVisibleRowsCsv(items: Campaign[], customerNameById: Record<string, string>) {
  const header = ['id', 'name', 'status', 'customer', 'budget', 'spend'];
  const lines = items.map((campaign) =>
    [
      campaign.id,
      campaign.name,
      campaign.status,
      customerNameById[campaign.customer_id] ?? campaign.customer_id,
      campaign.budget_limit,
      campaign.current_spend,
    ]
      .map((value) => `"${String(value).replaceAll('"', '""')}"`)
      .join(','),
  );
  const blob = new Blob([[header.join(','), ...lines].join('\n')], { type: 'text/csv;charset=utf-8' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = 'campaigns-export.csv';
  anchor.click();
  URL.revokeObjectURL(url);
}

export function CampaignsDirectory({
  items,
  total,
  limit,
  offset,
  customerOptions,
  customerNameById,
  metricsById,
  marginsById,
  columnWidthProbe,
  appliedStatus,
  appliedSort,
  appliedOrder,
  draftCustomerId,
  draftStatus,
  draftQ,
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
  onDraftQChange,
  onSearchApply,
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
  const [importOpen, setImportOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [overviewCampaign, setOverviewCampaign] = useState<CampaignWithMoneyDisplay | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(() => new Set());
  const [columnPrefs, setColumnPrefs] = useState<CampaignListColumnPrefs>(() =>
    loadCampaignListColumnPrefs(),
  );
  const [pageSizeDraft, setPageSizeDraft] = useState(String(limit));

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const createDisabled =
    creating || !customerId || !draftTemplateId || templatesLoading;
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, items.length);
  const rangeLabel = total === 0 ? '0 of 0' : `${rangeStart} - ${rangeEnd} of ${total}`;

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

  const headerSearch = useMemo(
    () => ({
      value: draftQ,
      onChange: onDraftQChange,
      onApply: onSearchApply,
      disabled: fetching,
      placeholder: 'Search',
    }),
    [draftQ, fetching, onDraftQChange, onSearchApply],
  );

  useTrackerHeaderSearchRegistration(headerSearch);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={8} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load campaigns" message={error.message} />;
  }

  return (
    <div className="campaigns-list-workspace flex min-h-full flex-col">
      <h1 className="sr-only">Campaigns</h1>

      <CampaignsListToolbar
        columnPrefs={columnPrefs}
        customerOptions={customerOptions}
        draftCustomerId={draftCustomerId}
        draftStatus={draftStatus}
        fetching={fetching}
        onColumnPrefsChange={setColumnPrefs}
        onCreateClick={() => onCreateSectionOpenChange(true)}
        onDraftCustomerIdChange={onDraftCustomerIdChange}
        onDraftStatusChange={onDraftStatusChange}
        onImportClick={() => setImportOpen(true)}
        onRefresh={onRefreshList}
        onWizardClick={() => setWizardOpen(true)}
      />

      <div className="flex min-h-0 flex-1 flex-col bg-white">
        <CampaignsListTable
          appliedOrder={appliedOrder}
          appliedSort={appliedSort}
          columnPrefs={columnPrefs}
          columnWidths={columnWidths}
          customerNameById={customerNameById}
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
          onColumnPrefsChange={setColumnPrefs}
          onColumnSort={onColumnSort}
          onColumnWidthCommit={handleColumnWidthCommit}
          onCreateClick={() => onCreateSectionOpenChange(true)}
          onSelectedIdsChange={setSelectedIds}
        />

        <div className="campaigns-list-workspace-footer">
          <div className="flex items-center gap-1">
            <button
              aria-label="Previous page"
              className="campaigns-list-workspace-icon-btn"
              disabled={fetching || !canGoPrev}
              type="button"
              onClick={() => onPageChange(Math.max(0, offset - limit))}
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </button>
            <button
              aria-label="Next page"
              className="campaigns-list-workspace-icon-btn"
              disabled={fetching || !canGoNext}
              type="button"
              onClick={() => onPageChange(offset + limit)}
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>

          <label className="flex items-center gap-1.5">
            <span>rows per page</span>
            <input
              className="campaigns-list-workspace-page-size tabular-nums"
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

          <span className="tabular-nums">{rangeLabel}</span>

          <div className="ml-auto">
            <DropdownExport
              disabled={fetching || items.length === 0}
              onExport={() => exportVisibleRowsCsv(items, customerNameById)}
            />
          </div>
        </div>
      </div>

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
                <SelectTrigger id="campaigns-template">
                  <SelectValue placeholder={templatesLoading ? 'Loading…' : 'Select template…'} />
                </SelectTrigger>
                <SelectContent>
                  {templates.map((template) => (
                    <SelectItem key={template.id} value={template.id}>
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
                placeholder="Optional display name…"
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
                placeholder="Optional override…"
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
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Import campaign</SheetTitle>
            <SheetDescription>Validate, migrate, or import a campaign bundle.</SheetDescription>
          </SheetHeader>
          <div className="mt-4">
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
          <div className="mt-4">
            <CampaignWizardPanel />
          </div>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function DropdownExport({
  disabled,
  onExport,
}: {
  disabled?: boolean;
  onExport: () => void;
}) {
  return (
    <div className="inline-flex overflow-hidden rounded-[3px] border border-[var(--campaigns-ws-border)]">
      <button
        className="campaigns-list-workspace-btn-secondary rounded-none border-0"
        disabled={disabled}
        type="button"
        onClick={onExport}
      >
        Export
      </button>
      <button
        aria-label="Export options"
        className="campaigns-list-workspace-icon-btn rounded-none border-0 border-l border-[var(--campaigns-ws-border)]"
        disabled={disabled}
        type="button"
      >
        <ChevronDown className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
