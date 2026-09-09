import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { DEFAULT_CLONE_OPTIONS } from '@/domains/campaigns/editor/campaign_clone_request';
import {
  campaignPanelError,
  CLONE_OPTION_FIELDS,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { useCampaignBulkCloneDialogWorkspace } from '@/domains/campaigns/list/use_campaign_bulk_clone_dialog_workspace';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { adminKit, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type CampaignBulkCloneDialogProps = {
  sourceCampaignIds: string[];
  customerId?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCloned?: () => void;
};

export function CampaignBulkCloneDialog({
  sourceCampaignIds,
  customerId,
  open,
  onOpenChange,
  onCloned,
}: CampaignBulkCloneDialogProps) {
  const {
    nameSuffix,
    setNameSuffix,
    cloneOptions,
    onCloneOptionChange,
    cloning,
    cloneError,
    results,
    onBulkClone,
  } = useCampaignBulkCloneDialogWorkspace({
    sourceCampaignIds,
    customerId,
    open,
    onCloned,
  });

  const failedCount = results?.filter((row) => !row.ok).length ?? 0;
  const successCount = results?.filter((row) => row.ok).length ?? 0;

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-lg" >
        <DialogHeader>
          <DialogTitle>Bulk clone campaigns</DialogTitle>
          <DialogDescription>
            Clone {sourceCampaignIds.length} selected campaign(s) with flow, postback config, and
            conversion mappings. Budget spend is reset on each clone.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="grid gap-4" >
          <div className={cn('grid', adminKit.fieldLabelGap)} >
            <Label htmlFor="bulk-clone-name-suffix">Name suffix</Label>
            <Input
              id="bulk-clone-name-suffix"
              disabled={cloning || successCount > 0}
              value={nameSuffix}
              onChange={(event) => setNameSuffix(event.target.value)}
            />
          </div>

          <div className="grid gap-3" >
            <p className={cn("m-0", adminTypography.label)}>Clone options</p>
            {CLONE_OPTION_FIELDS.map(({ field, label, description }) => {
              const inputId = `bulk-clone-option-${field}`;
              const checked = cloneOptions[field] ?? DEFAULT_CLONE_OPTIONS[field];
              return (
                <div className="grid gap-1" key={field}>
                  <div className="flex items-center gap-2" >
                    <Checkbox
                      checked={checked}
                      disabled={cloning || successCount > 0}
                      id={inputId}
                      onCheckedChange={(value) => onCloneOptionChange(field, value === true)}
                    />
                    <Label htmlFor={inputId}>{label}</Label>
                  </div>
                  <p className={adminTypography.captionPlain} >{description}</p>
                </div>
              );
            })}
          </div>

          {cloneError ? campaignPanelError(cloneError, 'Bulk clone failed') : null}
          {results && successCount > 0 ? (
            <p  className={adminTypography.bodyMuted} role="status">
              Cloned {successCount} campaign(s)
              {failedCount > 0 ? `; ${failedCount} failed` : ''}.
            </p>
          ) : null}
        </DialogBody>
        <DialogFooter className="gap-2" >
          <SecondaryActionButton type="button" onClick={() => onOpenChange(false)}>
            {successCount > 0 ? 'Close' : 'Cancel'}
          </SecondaryActionButton>
          <PrimaryActionButton
            disabled={sourceCampaignIds.length === 0 || successCount > 0}
            loading={cloning}
            type="button"
            onClick={onBulkClone}
          >
            {cloning ? 'Cloning...' : 'Clone selected'}
          </PrimaryActionButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
