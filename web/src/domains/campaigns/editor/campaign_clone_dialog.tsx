import { Link } from 'react-router-dom';

import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { DEFAULT_CLONE_OPTIONS } from '@/domains/campaigns/editor/campaign_clone_request';
import {
  campaignEditorActionsRowClass,
  campaignEditorSectionClass,
  campaignPanelError,
  CLONE_OPTION_FIELDS,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { useCampaignCloneDialogWorkspace } from '@/domains/campaigns/editor/use_campaign_clone_dialog_workspace';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { adminKit, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

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
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="gap-0 p-0 sm:max-w-2xl">
        <SheetHeader className="border-b border-border py-4 text-left">
          <SheetTitle>Clone campaign</SheetTitle>
          <SheetDescription>
            {campaignName ? (
              <>
                Source: <strong>{campaignName}</strong>
              </>
            ) : (
              'Clone the selected campaign.'
            )}
          </SheetDescription>
        </SheetHeader>

        <SheetBody className="grid gap-4 pb-8">
          <section className={cn(campaignEditorSectionClass, 'gap-3')}>
            <div className={cn('grid', adminKit.fieldLabelGap)}>
              <Label htmlFor="clone-name-suffix">Name suffix</Label>
              <Input
                id="clone-name-suffix"
                disabled={cloning || Boolean(clonedId)}
                value={nameSuffix}
                onChange={(event) => setNameSuffix(event.target.value)}
              />
              <p className={adminTypography.captionPlain}>
                Appended to the source name. Leave empty for the default &quot; (copy)&quot;.
              </p>
            </div>
          </section>

          <section className={cn(campaignEditorSectionClass, 'gap-3')}>
            <p className={cn('m-0', adminTypography.label)}>Clone options</p>
            <div className="grid gap-3">
              {CLONE_OPTION_FIELDS.map(({ field, label, description }) => {
                const inputId = `clone-option-${field}`;
                const checked = cloneOptions[field] ?? DEFAULT_CLONE_OPTIONS[field];

                return (
                  <div className="grid gap-1" key={field}>
                    <div className="flex items-center gap-2">
                      <Checkbox
                        checked={checked}
                        disabled={cloning || Boolean(clonedId)}
                        id={inputId}
                        onCheckedChange={(value) => onCloneOptionChange(field, value === true)}
                      />
                      <Label htmlFor={inputId}>{label}</Label>
                    </div>
                    <p className={adminTypography.captionPlain}>{description}</p>
                  </div>
                );
              })}
            </div>
          </section>

          {preview ? (
            <section className={cn(campaignEditorSectionClass, 'gap-2')}>
              <p className={cn('m-0', adminTypography.bodyMuted)}>
                Preview name: <strong className="text-foreground">{preview.name}</strong>
              </p>
            </section>
          ) : null}

          {previewError ? campaignPanelError(previewError, 'Preview failed') : null}
          {cloneError ? campaignPanelError(cloneError, 'Clone failed') : null}

          {clonedId ? (
            <p className={adminTypography.bodyMuted}>
              Created{' '}
              <Link className="text-primary hover:underline" to={`/campaigns/${clonedId}/edit`}>
                open cloned campaign
              </Link>
            </p>
          ) : null}

          <div className={campaignEditorActionsRowClass}>
            <SecondaryActionButton
              disabled={!campaignId || previewing || cloning || Boolean(clonedId)}
              type="button"
              onClick={onPreview}
            >
              {previewing ? 'Previewing...' : 'Preview'}
            </SecondaryActionButton>
            <PrimaryActionButton
              disabled={!campaignId || Boolean(clonedId)}
              loading={cloning}
              type="button"
              onClick={onClone}
            >
              {cloning ? 'Cloning...' : 'Clone'}
            </PrimaryActionButton>
          </div>
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
