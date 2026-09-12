// Campaign create UX: Quick create (createSectionOpen Dialog) is primary; Guided setup (wizardOpen
// Dialog) is secondary/advanced. open/set helpers in campaign_list_create_overlay.ts enforce mutex
// so only one class-C surface is open; bulkBusy blocks archive confirm while exportBusy is separate.
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { DirectoryMutationError } from '@/shell/directory_page_shell';
import { panelError } from '@/shell/panel_error';
import {
  Dialog,
  DialogBody,
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
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import type { CustomerComboboxOption } from '@/shell/customer_combobox';
import type { SelfServeCampaignTemplate } from '@/api/types';
import { CampaignBulkCloneDialog } from '@/domains/campaigns/list/campaign_bulk_clone_dialog';
import { CampaignBulkPatchDialog } from '@/domains/campaigns/list/campaign_bulk_patch_dialog';
import { CampaignCloneDialog } from '@/domains/campaigns/editor/campaign_clone_dialog';
import { CampaignImportPanel } from '@/domains/campaigns/editor/campaign_import_panel';
import { CampaignWizardPanel } from '@/domains/campaigns/editor/campaign_wizard_panel';
import type { CampaignImportPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_import_panel_workspace';
import type { CampaignWizardPanelWorkspace } from '@/domains/campaigns/editor/use_campaign_wizard_panel_workspace';

export type CampaignsDirectoryOverlaysProps = {
  actionError: Error | undefined;
  archiveOpen: boolean;
  bulkBusy: boolean;
  bulkCloneOpen: boolean;
  bulkPatchOpen: boolean;
  cloneOpen: boolean;
  createDisabled: boolean;
  createCustomerId: string;
  createSectionOpen: boolean;
  creating: boolean;
  customerId: string | undefined;
  customerNameById: Record<string, string>;
  customerOptions: CustomerComboboxOption[];
  customersLoading: boolean;
  draftBudgetLimitMicro: string;
  draftCreateName: string;
  draftTemplateId: string;
  importOpen: boolean;
  importPanelWorkspace: CampaignImportPanelWorkspace;
  onArchiveConfirm: () => void;
  onArchiveOpenChange: (open: boolean) => void;
  onBulkCloneOpenChange: (open: boolean) => void;
  onBulkPatchOpenChange: (open: boolean) => void;
  onBulkPatched: () => void;
  onCloneOpenChange: (open: boolean) => void;
  onBulkCloned: () => void;
  onCloned: () => void;
  selectedCampaignIds: string[];
  onCreateCampaign: () => void;
  onCreateSectionOpenChange: (open: boolean) => void;
  onDraftBudgetLimitMicroChange: (value: string) => void;
  onDraftCreateCustomerIdChange: (customerId: string) => void;
  onDraftCreateNameChange: (name: string) => void;
  onDraftTemplateIdChange: (templateId: string) => void;
  onImportOpenChange: (open: boolean) => void;
  onLoadTemplates: () => void;
  onWizardOpenChange: (open: boolean) => void;
  onWizardRefresh: () => void;
  selectedCampaignId: string | undefined;
  selectedCampaignName: string | undefined;
  selectedCount: number;
  templates: SelfServeCampaignTemplate[];
  templatesError: Error | undefined;
  templatesLoading: boolean;
  wizardOpen: boolean;
  wizardPanelWorkspace: CampaignWizardPanelWorkspace;
};

export function CampaignsDirectoryOverlays({
  actionError,
  archiveOpen,
  bulkBusy,
  bulkCloneOpen,
  bulkPatchOpen,
  cloneOpen,
  createDisabled,
  createCustomerId,
  createSectionOpen,
  creating,
  customerId,
  customerNameById,
  customerOptions,
  customersLoading,
  draftBudgetLimitMicro,
  draftCreateName,
  draftTemplateId,
  importOpen,
  importPanelWorkspace,
  onArchiveConfirm,
  onArchiveOpenChange,
  onBulkCloneOpenChange,
  onBulkPatchOpenChange,
  onBulkPatched,
  onCloneOpenChange,
  onBulkCloned,
  onCloned,
  onCreateCampaign,
  onCreateSectionOpenChange,
  onDraftBudgetLimitMicroChange,
  onDraftCreateCustomerIdChange,
  onDraftCreateNameChange,
  onDraftTemplateIdChange,
  onImportOpenChange,
  onLoadTemplates,
  onWizardOpenChange,
  onWizardRefresh,
  selectedCampaignId,
  selectedCampaignIds,
  selectedCampaignName,
  selectedCount,
  templates,
  templatesError,
  templatesLoading,
  wizardOpen,
  wizardPanelWorkspace,
}: CampaignsDirectoryOverlaysProps) {
  const effectiveCreateCustomerId = createCustomerId.trim() || customerId || '';
  const createFieldsDisabled = !effectiveCreateCustomerId;

  return (
    <>
      {actionError && !createSectionOpen ? <DirectoryMutationError error={actionError} /> : null}

      <Dialog open={createSectionOpen} onOpenChange={onCreateSectionOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Quick create campaign</DialogTitle>
            <DialogDescription>
              Pick a customer group and template for a fast campaign setup.
            </DialogDescription>
          </DialogHeader>

          <form
            onSubmit={(event) => {
              event.preventDefault();
              onCreateCampaign();
            }}
          >
            <div>
              <Label htmlFor="campaigns-create-customer">Customer group</Label>
              <Select
                disabled={customersLoading || customerOptions.length === 0}
                value={createCustomerId || undefined}
                onValueChange={onDraftCreateCustomerIdChange}
              >
                <SelectTrigger id="campaigns-create-customer">
                  <SelectValue
                    placeholder={
                      customersLoading
                        ? 'Loading...'
                        : customerOptions.length === 0
                          ? 'No customer groups'
                          : 'Select customer group...'
                    }
                  />
                </SelectTrigger>
                <SelectContent plain>
                  {customerOptions.map((customer) => (
                    <SelectItem key={customer.id} plain value={customer.id}>
                      {customer.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div>
              <Label htmlFor="campaigns-template">Template</Label>
              <Select
                disabled={createFieldsDisabled || templatesLoading}
                value={draftTemplateId}
                onValueChange={onDraftTemplateIdChange}
              >
                <SelectTrigger id="campaigns-template">
                  <SelectValue
                    placeholder={
                      createFieldsDisabled
                        ? 'Select customer group first...'
                        : templatesLoading
                          ? 'Loading...'
                          : templates.length === 0
                            ? 'No templates'
                            : 'Select template...'
                    }
                  />
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

            <div>
              <Label htmlFor="campaigns-create-name">Name</Label>
              <Input
                id="campaigns-create-name"
                disabled={createFieldsDisabled}
                placeholder="Optional display name..."
                value={draftCreateName}
                onChange={(event) => onDraftCreateNameChange(event.target.value)}
              />
            </div>

            <div>
              <Label htmlFor="campaigns-budget-micro">Budget (micro)</Label>
              <Input
                id="campaigns-budget-micro"
                disabled={createFieldsDisabled}
                inputMode="numeric"
                placeholder="Optional override..."
                value={draftBudgetLimitMicro}
                onChange={(event) => onDraftBudgetLimitMicroChange(event.target.value)}
              />
            </div>

            {templatesError ? panelError(templatesError, 'Could not load templates') : null}
            {actionError ? <DirectoryMutationError error={actionError} /> : null}
            {effectiveCreateCustomerId &&
            !templatesLoading &&
            templates.length === 0 &&
            !templatesError ? (
              <p>
                No templates for{' '}
                {customerNameById[effectiveCreateCustomerId] ?? effectiveCreateCustomerId}.
              </p>
            ) : null}

            <DialogFooter>
              <SecondaryActionButton
                disabled={createFieldsDisabled}
                loading={templatesLoading}
                onClick={onLoadTemplates}
                type="button"
              >
                Reload templates
              </SecondaryActionButton>
              <PrimaryActionButton disabled={createDisabled} loading={creating} type="submit">
                Quick create
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

      <CampaignBulkCloneDialog
        customerId={customerId}
        open={bulkCloneOpen}
        sourceCampaignIds={selectedCampaignIds}
        onCloned={onBulkCloned}
        onOpenChange={onBulkCloneOpenChange}
      />

      <CampaignBulkPatchDialog
        campaignIds={selectedCampaignIds}
        open={bulkPatchOpen}
        onOpenChange={onBulkPatchOpenChange}
        onPatched={onBulkPatched}
      />

      <Dialog open={archiveOpen} onOpenChange={onArchiveOpenChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Archive campaigns</DialogTitle>
            <DialogDescription>
              Archive {selectedCount} selected campaign(s)? They can be filtered under Archived
              status.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <SecondaryActionButton type="button" onClick={() => onArchiveOpenChange(false)}>
              Cancel
            </SecondaryActionButton>
            <PrimaryActionButton loading={bulkBusy} type="button" onClick={onArchiveConfirm}>
              Archive
            </PrimaryActionButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Sheet onOpenChange={onImportOpenChange} open={importOpen}>
        <SheetContent className="gap-0 p-0 sm:max-w-2xl">
          <SheetHeader className="border-b border-border px-6 py-4 text-left">
            <SheetTitle>Import campaign</SheetTitle>
            <SheetDescription>Validate, migrate, or import a campaign bundle.</SheetDescription>
          </SheetHeader>
          <SheetBody className="grid gap-4 pb-8">
            <CampaignImportPanel workspace={importPanelWorkspace} />
          </SheetBody>
        </SheetContent>
      </Sheet>

      <Dialog onOpenChange={onWizardOpenChange} open={wizardOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Guided setup</DialogTitle>
            <DialogDescription>
              Step-by-step campaign setup with traffic, flow, and budget.
            </DialogDescription>
          </DialogHeader>
          <DialogBody>
            <CampaignWizardPanel workspace={wizardPanelWorkspace} />
          </DialogBody>
        </DialogContent>
      </Dialog>
    </>
  );
}
