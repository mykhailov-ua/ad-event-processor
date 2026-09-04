import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { ErrorBlock } from '@/shell/error_block';
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
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { CampaignStatsQuery, SelfServeCampaignTemplate } from '@/api/types';
import type { CampaignWithMoneyDisplay } from '@/domains/campaigns/list/campaign_metrics_shared';
import { CampaignCloneDialog } from '@/domains/campaigns/editor/campaign_clone_dialog';
import { CampaignImportPanel } from '@/domains/campaigns/editor/campaign_import_panel';
import { CampaignOverviewSheet } from '@/domains/campaigns/list/campaign_overview_sheet';
import { CampaignWizardPanel } from '@/domains/campaigns/editor/campaign_wizard_panel';
import { CampaignListResetWorkspaceDialog } from '@/domains/campaigns/list/campaign_list_reset_workspace_dialog';

export type CampaignsDirectoryOverlaysProps = {
  actionError: Error | undefined;
  archiveOpen: boolean;
  bulkBusy: boolean;
  cloneOpen: boolean;
  createDisabled: boolean;
  createSectionOpen: boolean;
  creating: boolean;
  customerId: string | undefined;
  customerNameById: Record<string, string>;
  customerOptions: CustomerComboboxOption[];
  draftBudgetLimitMicro: string;
  draftCreateName: string;
  draftTemplateId: string;
  importOpen: boolean;
  onArchiveConfirm: () => void;
  onArchiveOpenChange: (open: boolean) => void;
  onCloneOpenChange: (open: boolean) => void;
  onCloned: () => void;
  onCreateCampaign: () => void;
  onCreateSectionOpenChange: (open: boolean) => void;
  onDraftBudgetLimitMicroChange: (value: string) => void;
  onDraftCreateNameChange: (name: string) => void;
  onDraftTemplateIdChange: (templateId: string) => void;
  onImportOpenChange: (open: boolean) => void;
  onLoadTemplates: () => void;
  onOverviewOpenChange: (open: boolean) => void;
  onResetWorkspaceConfirm: () => void;
  onResetWorkspaceOpenChange: (open: boolean) => void;
  onWizardOpenChange: (open: boolean) => void;
  onWizardRefresh: () => void;
  overviewCampaign: CampaignWithMoneyDisplay | null;
  resetWorkspaceOpen: boolean;
  selectedCampaignId: string | undefined;
  selectedCampaignName: string | undefined;
  selectedCount: number;
  statsQuery: CampaignStatsQuery;
  templates: SelfServeCampaignTemplate[];
  templatesError: Error | undefined;
  templatesLoading: boolean;
  wizardOpen: boolean;
};

export function CampaignsDirectoryOverlays({
  actionError,
  archiveOpen,
  bulkBusy,
  cloneOpen,
  createDisabled,
  createSectionOpen,
  creating,
  customerId,
  customerNameById,
  customerOptions,
  draftBudgetLimitMicro,
  draftCreateName,
  draftTemplateId,
  importOpen,
  onArchiveConfirm,
  onArchiveOpenChange,
  onCloneOpenChange,
  onCloned,
  onCreateCampaign,
  onCreateSectionOpenChange,
  onDraftBudgetLimitMicroChange,
  onDraftCreateNameChange,
  onDraftTemplateIdChange,
  onImportOpenChange,
  onLoadTemplates,
  onOverviewOpenChange,
  onResetWorkspaceConfirm,
  onResetWorkspaceOpenChange,
  onWizardOpenChange,
  onWizardRefresh,
  overviewCampaign,
  resetWorkspaceOpen,
  selectedCampaignId,
  selectedCampaignName,
  selectedCount,
  statsQuery,
  templates,
  templatesError,
  templatesLoading,
  wizardOpen,
}: CampaignsDirectoryOverlaysProps) {
  return (
    <>
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
        campaignName={selectedCampaignName}
        open={cloneOpen}
        onCloned={onCloned}
        onOpenChange={onCloneOpenChange}
      />

      <Dialog open={archiveOpen} onOpenChange={onArchiveOpenChange}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Archive campaigns</DialogTitle>
            <DialogDescription>
              Archive {selectedCount} selected campaign(s)? They can be filtered under Archived
              status.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2">
            <SecondaryActionButton type="button" onClick={() => onArchiveOpenChange(false)}>
              Cancel
            </SecondaryActionButton>
            <PrimaryActionButton loading={bulkBusy} type="button" onClick={onArchiveConfirm}>
              Archive
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <CampaignListResetWorkspaceDialog
        open={resetWorkspaceOpen}
        onConfirm={onResetWorkspaceConfirm}
        onOpenChange={onResetWorkspaceOpenChange}
      />

      <CampaignOverviewSheet
        campaign={overviewCampaign}
        customerName={
          overviewCampaign
            ? customerNameById[overviewCampaign.customer_id] ?? overviewCampaign.customer_id
            : ''
        }
        onOpenChange={onOverviewOpenChange}
        open={overviewCampaign != null}
        statsQuery={statsQuery}
      />

      <Sheet onOpenChange={onImportOpenChange} open={importOpen}>
        <SheetContent className="flex h-full w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-2xl">
          <SheetHeader className="shrink-0 border-b border-border px-6 py-4 text-left">
            <SheetTitle>Import campaign</SheetTitle>
            <SheetDescription>Validate, migrate, or import a campaign bundle.</SheetDescription>
          </SheetHeader>
          <div className="ui-scrollbar min-h-0 flex-1 overflow-y-auto px-6 py-4 pb-8">
            <CampaignImportPanel />
          </div>
        </SheetContent>
      </Sheet>

      <Sheet onOpenChange={onWizardOpenChange} open={wizardOpen}>
        <SheetContent className="w-full overflow-y-auto sm:max-w-2xl">
          <SheetHeader>
            <SheetTitle>Campaign wizard</SheetTitle>
            <SheetDescription>Guided setup for a new campaign.</SheetDescription>
          </SheetHeader>
          <div className="mt-6">
            <CampaignWizardPanel
              customerOptions={customerOptions}
              onCampaignCreated={onWizardRefresh}
            />
          </div>
        </SheetContent>
      </Sheet>
    </>
  );
}
