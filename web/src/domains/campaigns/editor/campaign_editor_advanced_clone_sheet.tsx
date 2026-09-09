import { Link } from 'react-router-dom';

import type { CloneCampaignOptions, CloneCampaignPreview } from '@/api/types';
import type { Campaign } from '@/api/types';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Sheet, SheetBody, SheetContent, SheetHeader, SheetTitle } from '@/components/ui/sheet';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { formatCampaignJsonKey } from '@/domains/campaigns/editor/campaign_json_labels';
import {
  CLONE_OPTION_FIELDS,
  campaignEditorActionsRowClass,
  campaignEditorSectionClass,
  editorApiErrorBlock,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

type CampaignEditorAdvancedCloneSheetProps = {
  campaign: Campaign;
  cloneOpen: boolean;
  onCloneOpenChange: (open: boolean) => void;
  cloneNameSuffix: string;
  onCloneNameSuffixChange: (value: string) => void;
  cloneOptions: CloneCampaignOptions;
  onCloneOptionChange: (field: keyof CloneCampaignOptions, checked: boolean) => void;
  clonePreviewing: boolean;
  cloning: boolean;
  fetching: boolean;
  clonePreview: CloneCampaignPreview | undefined;
  clonePreviewError: Error | undefined;
  cloneError: Error | undefined;
  cloneSuccess: boolean;
  clonedCampaignId: string | undefined;
  onClonePreview: () => void;
  onCloneExecute: () => void;
};

export function CampaignEditorAdvancedCloneSheet({
  campaign,
  cloneOpen,
  onCloneOpenChange,
  cloneNameSuffix,
  onCloneNameSuffixChange,
  cloneOptions,
  onCloneOptionChange,
  clonePreviewing,
  cloning,
  fetching,
  clonePreview,
  clonePreviewError,
  cloneError,
  cloneSuccess,
  clonedCampaignId,
  onClonePreview,
  onCloneExecute,
}: CampaignEditorAdvancedCloneSheetProps) {
  return (
    <Sheet onOpenChange={onCloneOpenChange} open={cloneOpen}>
      <SheetContent >
        <SheetHeader >
          <SheetTitle>Clone campaign</SheetTitle>
        </SheetHeader>
        <SheetBody >
          <section >
            <div >
              <Label htmlFor="campaign-clone-name-suffix">Clone name suffix</Label>
              <Input
                id="campaign-clone-name-suffix"
                value={cloneNameSuffix}
                disabled={clonePreviewing || cloning || fetching}
                placeholder=" (copy)"
                onChange={(event) => onCloneNameSuffixChange(event.target.value)}
              />
              <p >
                Leave empty for the default &quot;{campaign.name} (copy)&quot;. Enter a suffix such
                as &quot; - v2&quot; to append to the source name.
              </p>
            </div>
          </section>

          <section >
            <p >Clone options</p>
            <div >
              {CLONE_OPTION_FIELDS.map(({ field, label, description }) => {
                const inputId = `campaign-clone-option-${field}`;
                const defaultChecked = field === 'reset_spend' ? false : true;
                const checked = cloneOptions[field] ?? defaultChecked;

                return (
                  <div key={field}>
                    <div >
                      <Checkbox
                        checked={checked}
                        disabled={clonePreviewing || cloning || fetching}
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

          <div >
            <Button
              type="button"
              variant="secondary"
              disabled={clonePreviewing || cloning || fetching}
              onClick={onClonePreview}
            >
              {clonePreviewing ? 'Previewing...' : 'Preview clone'}
            </Button>
            <Button
              type="button"
              disabled={clonePreviewing || cloning || fetching}
              onClick={onCloneExecute}
            >
              {cloning ? 'Creating clone...' : 'Create clone'}
            </Button>
          </div>

          {cloneError
            ? editorApiErrorBlock(cloneError, 'Clone unavailable', 'Could not create clone')
            : null}

          {cloneSuccess && clonedCampaignId ? (
            <p >
              Clone created.{' '}
              <Link
               
                to={`/campaigns/${clonedCampaignId}/edit`}
              >
                Open cloned campaign
              </Link>
            </p>
          ) : null}

          {clonePreviewError
            ? editorApiErrorBlock(
                clonePreviewError,
                'Clone preview unavailable',
                'Could not preview clone'
              )
            : null}

          {clonePreview ? (
            <JsonPayloadView
              formatColumn={formatCampaignJsonKey}
              formatKey={formatCampaignJsonKey}
              payload={clonePreview}
            />
          ) : null}
        </SheetBody>
      </SheetContent>
    </Sheet>
  );
}
