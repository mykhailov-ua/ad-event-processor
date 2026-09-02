import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { CAMPAIGN_LIST_WORKSPACE_RESET_ITEMS } from '@/domains/campaigns/list/campaign_list_workspace_prefs';

export type CampaignListResetWorkspaceDialogProps = {
  open: boolean;
  busy?: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
};

export function CampaignListResetWorkspaceDialog({
  open,
  busy = false,
  onOpenChange,
  onConfirm,
}: CampaignListResetWorkspaceDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md p-0">
        <div className="admin-reset-dialog">
          <DialogHeader className="admin-reset-dialog__header">
            <DialogTitle>Reset campaign list view</DialogTitle>
            <DialogDescription>
              Restore saved layout preferences for this page. Filters, sort, and pagination are not
              changed.
            </DialogDescription>
          </DialogHeader>

          <ul className="admin-reset-list">
            {CAMPAIGN_LIST_WORKSPACE_RESET_ITEMS.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>

          <DialogFooter className="admin-reset-dialog__footer">
            <SecondaryActionButton disabled={busy} type="button" onClick={() => onOpenChange(false)}>
              Cancel
            </SecondaryActionButton>
            <PrimaryActionButton loading={busy} type="button" onClick={onConfirm}>
              Reset view
            </PrimaryActionButton>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
