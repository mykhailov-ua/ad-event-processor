import { Link } from 'react-router-dom';

import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
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
import { DEFAULT_CLONE_OPTIONS } from '@/domains/campaigns/editor/campaign_clone_request';
import {
  campaignPanelError,
  CLONE_OPTION_FIELDS,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { useCampaignCloneDialogWorkspace } from '@/domains/campaigns/editor/use_campaign_clone_dialog_workspace';

export type CampaignCloneDialogProps = {
  campaignId: string | undefined;
  campaignName?: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCloned?: (newCampaignId: string) => void;
};

export function CampaignCloneDialog({
  campaignId,
  campaignName,
  open,
  onOpenChange,
  onCloned,
}: CampaignCloneDialogProps) {
  const {
    nameSuffix,
    setNameSuffix,
    cloneOptions,
    onCloneOptionChange,
    preview,
    previewError,
    previewing,
    cloning,
    cloneError,
    clonedId,
    onPreview,
    onClone,
  } = useCampaignCloneDialogWorkspace({ campaignId, open, onCloned });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Clone campaign</DialogTitle>
          <DialogDescription>
            {campaignName ? (
              <>
                Source: <strong>{campaignName}</strong>
              </>
            ) : (
              'Clone the selected campaign.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4">
          <div className="grid gap-1">
            <Label htmlFor="clone-name-suffix">Name suffix</Label>
            <Input
              id="clone-name-suffix"
              value={nameSuffix}
              onChange={(event) => setNameSuffix(event.target.value)}
            />
          </div>

          <div className="grid gap-3">
            {CLONE_OPTION_FIELDS.map(({ field, label }) => (
              <label key={field} className="flex items-center gap-2">
                <Checkbox
                  checked={cloneOptions[field] ?? DEFAULT_CLONE_OPTIONS[field]}
                  onCheckedChange={(checked) => onCloneOptionChange(field, checked === true)}
                />
                <span>{label}</span>
              </label>
            ))}
          </div>

          {preview ? (
            <p className="text-sm text-muted-foreground">
              Preview name: <strong>{preview.name}</strong>
            </p>
          ) : null}
          {previewError ? campaignPanelError(previewError, 'Preview failed') : null}
          {cloneError ? campaignPanelError(cloneError, 'Clone failed') : null}
          {clonedId ? (
            <p>
              Created{' '}
              <Button asChild type="button" variant="link">
                <Link to={`/campaigns/${clonedId}/edit`}>{clonedId}</Link>
              </Button>
            </p>
          ) : null}
        </div>

        <DialogFooter className="gap-2">
          <Button
            disabled={!campaignId || previewing}
            type="button"
            variant="outline"
            onClick={onPreview}
          >
            {previewing ? 'Previewing...' : 'Preview'}
          </Button>
          <Button
            disabled={!campaignId || cloning || Boolean(clonedId)}
            type="button"
            onClick={onClone}
          >
            {cloning ? 'Cloning...' : 'Clone'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
