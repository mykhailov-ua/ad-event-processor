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
import { adminKit } from '@/lib/admin_kit';
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
      <SheetContent >
        <SheetHeader >
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

        <SheetBody >
          <section >
            <div >
              <Label htmlFor="clone-name-suffix">Name suffix</Label>
              <Input
                id="clone-name-suffix"
                disabled={cloning || Boolean(clonedId)}
                value={nameSuffix}
                onChange={(event) => setNameSuffix(event.target.value)}
              />
              <p >
                Appended to the source name. Leave empty for the default &quot; (copy)&quot;.
              </p>
            </div>
          </section>

          <section >
            <p >Clone options</p>
            <div >
              {CLONE_OPTION_FIELDS.map(({ field, label, description }) => {
                const inputId = `clone-option-${field}`;
                const checked = cloneOptions[field] ?? DEFAULT_CLONE_OPTIONS[field];

                return (
                  <div key={field}>
                    <div >
                      <Checkbox
                        checked={checked}
                        disabled={cloning || Boolean(clonedId)}
                        id={inputId}
                        onCheckedChange={(value) => onCloneOptionChange(field, value === true)}
                      />
                      <Label htmlFor={inputId}>{label}</Label>
                    </div>
                    <p >{description}</p>
                  </div>
                );
              })}
            </div>
          </section>

          {preview ? (
            <section >
              <p >
                Preview name: <strong >{preview.name}</strong>
              </p>
            </section>
          ) : null}

          {previewError ? campaignPanelError(previewError, 'Preview failed') : null}
          {cloneError ? campaignPanelError(cloneError, 'Clone failed') : null}

          {clonedId ? (
            <p >
              Created{' '}
              <Link  to={`/campaigns/${clonedId}/edit`}>
                open cloned campaign
              </Link>
            </p>
          ) : null}

          <div >
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
