import { useState } from 'react';
import { Plus, Upload, Wand2 } from 'lucide-react';
import { Link } from 'react-router-dom';

import { FilterApplyButton, FilterResetButton, PrimaryActionButton, SecondaryActionButton } from '@/components/system/action_buttons';
import { AppliedCustomerBanner } from '@/components/system/applied_customer_banner';
import {
  DirectoryTable,
  DirectoryTableHead,
  SortableTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/components/system/directory_table';
import { DirectoryListMeta } from '@/components/system/directory_list_meta';
import {
  DirectoryFilterForm,
  FilterField,
  FilterPanel,
} from '@/components/system/filter_panel';
import { PageChrome } from '@/components/system/page_chrome';
import { PaginationPrevNext } from '@/components/system/pagination_prev_next';
import { RowActionsMenu } from '@/components/system/row_actions_menu';
import { ToggleChipGroup } from '@/components/system/toggle_chip_group';
import { CustomerCombobox, type CustomerComboboxOption } from '@/components/system/customer_combobox';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DropdownMenuItem, DropdownMenuSeparator } from '@/components/ui/dropdown-menu';
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
import { EmptyState } from '@/components/system/empty_state';
import { ErrorBlock } from '@/components/system/error_block';
import { PageSkeleton } from '@/components/system/page_skeleton';
import type { CampaignStatusTotals } from '@/api/campaigns_api';
import type { Campaign, SelfServeCampaignTemplate } from '@/api/types';
import { CampaignMetricsPopover } from '@/domains/campaigns/campaign_metrics_popover';
import {
  campaignForecastHref,
  campaignFraudHref,
  type CampaignWithMoneyDisplay,
} from '@/domains/campaigns/campaign_metrics_shared';
import { CampaignOverviewSheet } from '@/domains/campaigns/campaign_overview_sheet';
import { CampaignImportPanel } from '@/domains/campaigns/campaign_import_panel';
import { CampaignStatusBadge } from '@/domains/campaigns/campaign_status_badge';
import { CampaignWizardPanel } from '@/domains/campaigns/campaign_wizard_panel';
import { displayTimestamp } from '@/lib/display';
import { listPageRange } from '@/lib/list_page_stats';

export type CampaignSortField = 'name' | 'updated_at' | 'spend' | 'budget_limit';
export type SortOrder = 'asc' | 'desc';
export type CampaignStatusFilter = '' | 'ACTIVE' | 'PAUSED' | 'ARCHIVED';

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
  actingCampaignId: string | undefined;
  actionError: Error | undefined;
  onDraftCustomerIdChange: (customerId: string) => void;
  onDraftStatusChange: (status: CampaignStatusFilter) => void;
  onDraftQChange: (q: string) => void;
  onApplyFilters: () => void;
  onResetFilters: () => void;
  onClearCustomerScope: () => void;
  onStatusFilter: (status: CampaignStatusFilter) => void;
  onColumnSort: (field: CampaignSortField) => void;
  onPageChange: (nextOffset: number) => void;
  onDraftTemplateIdChange: (templateId: string) => void;
  onDraftCreateNameChange: (name: string) => void;
  onDraftBudgetLimitMicroChange: (value: string) => void;
  onLoadTemplates: () => void;
  onCreateCampaign: () => void;
  onPauseCampaign: (campaignId: string) => void;
  onResumeCampaign: (campaignId: string) => void;
  onArchiveCampaign: (campaignId: string) => void;
};

function isPausedStatus(status: string): boolean {
  return status.toUpperCase() === 'PAUSED';
}

function isArchivedStatus(status: string): boolean {
  return status.toUpperCase() === 'ARCHIVED';
}

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
  appliedCustomerId,
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
  actingCampaignId,
  actionError,
  onDraftCustomerIdChange,
  onDraftStatusChange,
  onDraftQChange,
  onApplyFilters,
  onResetFilters,
  onClearCustomerScope,
  onStatusFilter,
  onColumnSort,
  onPageChange,
  onDraftTemplateIdChange,
  onDraftCreateNameChange,
  onDraftBudgetLimitMicroChange,
  onLoadTemplates,
  onCreateCampaign,
  onPauseCampaign,
  onResumeCampaign,
  onArchiveCampaign,
}: CampaignsDirectoryProps) {
  const [importOpen, setImportOpen] = useState(false);
  const [wizardOpen, setWizardOpen] = useState(false);
  const [archiveCampaignId, setArchiveCampaignId] = useState<string | undefined>();
  const [overviewCampaign, setOverviewCampaign] = useState<CampaignWithMoneyDisplay | null>(null);

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton variant="directory" columns={6} />;
  }

  if (error && !hasSnapshot) {
    return <ErrorBlock title="Could not load campaigns" message={error.message} />;
  }

  const canGoPrev = offset > 0;
  const canGoNext = offset + limit < total;
  const createDisabled =
    creating || !customerId || !draftTemplateId || templatesLoading;
  const scopedCustomerName =
    customerNameById[appliedCustomerId] ?? appliedCustomerId;
  const { rangeStart, rangeEnd } = listPageRange(total, limit, offset, items.length);
  const rangeLabel =
    total === 0 ? 'No campaigns' : `Showing ${rangeStart}–${rangeEnd} of ${total}`;
  const statusChipOptions: Array<{
    value: CampaignStatusFilter;
    label: string;
    count?: number;
  }> = [
    { value: '', label: 'All', count: statusTotals?.total },
    { value: 'ACTIVE', label: 'Active', count: statusTotals?.active },
    { value: 'PAUSED', label: 'Paused', count: statusTotals?.paused },
    { value: 'ARCHIVED', label: 'Archived', count: statusTotals?.archived },
  ];

  return (
    <PageChrome
      title="Campaigns"
      description="Manage budgets, pacing, and delivery across customers."
      actions={
        <>
          <Button onClick={() => onCreateSectionOpenChange(true)} type="button">
            <Plus className="h-4 w-4" />
            Create campaign
          </Button>
          <Button onClick={() => setImportOpen(true)} type="button" variant="outline">
            <Upload className="h-4 w-4" />
            Import
          </Button>
          <Button onClick={() => setWizardOpen(true)} type="button" variant="outline">
            <Wand2 className="h-4 w-4" />
            Wizard
          </Button>
        </>
      }
    >
      {appliedCustomerId ? (
        <AppliedCustomerBanner
          customerId={appliedCustomerId}
          customerName={scopedCustomerName}
          onClear={onClearCustomerScope}
        />
      ) : null}

      <FilterPanel>
        <DirectoryFilterForm
          layout="directory"
          onSubmit={(event) => {
            event.preventDefault();
            onApplyFilters();
          }}
        >
          <FilterField htmlFor="campaigns-customer" label="Customer">
            <CustomerCombobox
              id="campaigns-customer"
              disabled={fetching}
              loading={customersLoading}
              options={customerOptions}
              value={draftCustomerId}
              onValueChange={onDraftCustomerIdChange}
            />
          </FilterField>

          <FilterField htmlFor="campaigns-status" label="Status">
            <Select
              value={draftStatus || 'all'}
              onValueChange={(value) =>
                onDraftStatusChange(value === 'all' ? '' : (value as CampaignStatusFilter))
              }
            >
              <SelectTrigger id="campaigns-status" className="h-9 w-full text-sm">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All</SelectItem>
                <SelectItem value="ACTIVE">Active</SelectItem>
                <SelectItem value="PAUSED">Paused</SelectItem>
                <SelectItem value="ARCHIVED">Archived</SelectItem>
              </SelectContent>
            </Select>
          </FilterField>

          <FilterField htmlFor="campaigns-q" label="Search" wide>
            <Input
              id="campaigns-q"
              className="h-9 text-sm"
              placeholder="Name or ID…"
              value={draftQ}
              onChange={(event) => onDraftQChange(event.target.value)}
            />
          </FilterField>

          <FilterApplyButton disabled={fetching}>Apply</FilterApplyButton>

          <FilterResetButton disabled={fetching} onClick={onResetFilters}>
            Reset
          </FilterResetButton>

          <PaginationPrevNext
            canGoPrev={canGoPrev}
            canGoNext={canGoNext}
            disabled={fetching}
            onPrev={() => onPageChange(Math.max(0, offset - limit))}
            onNext={() => onPageChange(offset + limit)}
          />
        </DirectoryFilterForm>
      </FilterPanel>

      <div className="grid gap-3">
        <DirectoryListMeta>{rangeLabel}</DirectoryListMeta>

        <ToggleChipGroup
          countsLoading={statusTotalsLoading}
          onChange={onStatusFilter}
          options={statusChipOptions}
          value={appliedStatus}
        />
      </div>

      <div aria-atomic="true" aria-live="polite">
        {items.length === 0 ? (
          filtersActive ? (
            <EmptyState
              variant="no-results"
              title="No campaigns"
              description="No campaigns match the current filters."
              actionLabel="Clear filters"
              onAction={onResetFilters}
            />
          ) : (
            <EmptyState
              variant="blank-slate"
              title="No campaigns"
              description="Create a campaign to start tracking spend and delivery."
              actionLabel="Create campaign"
              onAction={() => onCreateSectionOpenChange(true)}
            />
          )
        ) : (
          <DirectoryTable scrollable>
            <TableHeader className="sticky top-0 z-10 bg-card/95 backdrop-blur-sm [&_tr]:border-b [&_tr]:border-border/40">
              <TableRow>
                <SortableTableHead
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Name"
                  onSort={(field) => onColumnSort(field as CampaignSortField)}
                  sortField="name"
                />
                <DirectoryTableHead>Status</DirectoryTableHead>
                <SortableTableHead
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Budget used"
                  onSort={(field) => onColumnSort(field as CampaignSortField)}
                  sortField="spend"
                />
                <DirectoryTableHead>Customer</DirectoryTableHead>
                <SortableTableHead
                  activeOrder={appliedOrder}
                  activeSort={appliedSort}
                  label="Updated"
                  onSort={(field) => onColumnSort(field as CampaignSortField)}
                  sortField="updated_at"
                />
                <DirectoryTableHead className="w-[4.5rem]">Actions</DirectoryTableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {items.map((campaign) => {
                const row = campaign as CampaignWithMoneyDisplay;
                const archived = isArchivedStatus(campaign.status);
                const paused = isPausedStatus(campaign.status);
                const acting = actingCampaignId === campaign.id;
                const customerName =
                  customerNameById[campaign.customer_id] ?? campaign.customer_id;

                return (
                  <TableRow key={campaign.id}>
                    <TableCell className="font-medium">
                      <Link
                        className="text-primary hover:underline"
                        to={`/campaigns/${campaign.id}/edit`}
                      >
                        {campaign.name}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <CampaignStatusBadge campaign={campaign} className="px-1.5 py-0 font-normal" />
                    </TableCell>
                    <TableCell>
                      <CampaignMetricsPopover
                        campaign={row}
                        onOpenOverview={setOverviewCampaign}
                      />
                    </TableCell>
                    <TableCell>
                      <Link
                        className="text-primary hover:underline"
                        to={`/customers/${campaign.customer_id}`}
                      >
                        {customerName}
                      </Link>
                    </TableCell>
                    <TableCell>
                      {displayTimestamp(campaign.updated_at, campaign.updated_at_display)}
                    </TableCell>
                    <TableCell>
                      <RowActionsMenu ariaLabel={`Actions for ${campaign.name}`} disabled={acting}>
                        <DropdownMenuItem
                          onClick={() => setOverviewCampaign(row)}
                        >
                          Overview
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem asChild>
                          <Link to={`/campaigns/${campaign.id}/edit`}>Edit</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem asChild>
                          <Link to={`/dashboards/campaign/${campaign.id}`}>Dashboard</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem asChild>
                          <Link to={campaignForecastHref(campaign.customer_id)}>Forecast</Link>
                        </DropdownMenuItem>
                        <DropdownMenuItem asChild>
                          <Link to={campaignFraudHref(campaign.id, campaign.customer_id)}>
                            Fraud explain
                          </Link>
                        </DropdownMenuItem>
                        {!archived ? (
                          <>
                            <DropdownMenuSeparator />
                            {!paused ? (
                              <DropdownMenuItem
                                disabled={acting}
                                onClick={() => onPauseCampaign(campaign.id)}
                              >
                                Pause
                              </DropdownMenuItem>
                            ) : (
                              <DropdownMenuItem
                                disabled={acting}
                                onClick={() => onResumeCampaign(campaign.id)}
                              >
                                Resume
                              </DropdownMenuItem>
                            )}
                            <DropdownMenuItem
                              className="text-destructive focus:text-destructive"
                              disabled={acting}
                              onClick={() => setArchiveCampaignId(campaign.id)}
                            >
                              Archive
                            </DropdownMenuItem>
                          </>
                        ) : null}
                      </RowActionsMenu>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </DirectoryTable>
        )}
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
                'Set customer_id in the URL or session to create a campaign.'
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
                <SelectTrigger id="campaigns-template" className="h-9 w-full text-sm">
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
                className="h-9 text-sm"
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
                className="h-9 text-sm"
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

      <Dialog
        onOpenChange={(open) => {
          if (!open) {
            setArchiveCampaignId(undefined);
          }
        }}
        open={Boolean(archiveCampaignId)}
      >
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Archive campaign</DialogTitle>
            <DialogDescription>
              Archived campaigns stop delivery. You can restore status from the campaign editor later.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <SecondaryActionButton onClick={() => setArchiveCampaignId(undefined)} type="button">
              Cancel
            </SecondaryActionButton>
            <Button
              className="h-9 text-sm"
              disabled={actingCampaignId === archiveCampaignId}
              loading={actingCampaignId === archiveCampaignId}
              shape="pill"
              type="button"
              variant="destructive"
              onClick={() => {
                if (archiveCampaignId) {
                  onArchiveCampaign(archiveCampaignId);
                  setArchiveCampaignId(undefined);
                }
              }}
            >
              Archive campaign
            </Button>
          </DialogFooter>
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
    </PageChrome>
  );
}
