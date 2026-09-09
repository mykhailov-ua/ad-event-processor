import type {
  CampaignPublishBlockedError,
  CampaignPublishCheck,
  CampaignValidateResponse,
} from '@/api/types';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Checkbox } from '@/components/ui/checkbox';
import { Label } from '@/components/ui/label';
import {
  campaignEditorSectionClass,
  editorApiErrorBlock,
  FieldErrorsPanel,
  StringList,
  ValidityBadge,
} from '@/domains/campaigns/editor/campaign_editor_shared';
import { cn } from '@/lib/utils';
import { adminTypography } from '@/lib/admin_kit';

type CampaignEditorAdvancedPublishSectionProps = {
  checking: boolean;
  validating: boolean;
  publishing: boolean;
  forcePublish: boolean;
  gateBusy: boolean;
  fetching: boolean;
  saving: boolean;
  publishCheck: CampaignPublishCheck | undefined;
  validateResult: CampaignValidateResponse | undefined;
  publishBlocked: CampaignPublishBlockedError | undefined;
  publishSuccess: boolean;
  publishCheckError: Error | undefined;
  validateError: Error | undefined;
  publishError: Error | undefined;
  onForcePublishChange: (value: boolean) => void;
  onCheckPublish: () => void;
  onValidateChanges: () => void;
  onPublish: () => void;
};

export function CampaignEditorAdvancedPublishSection({
  checking,
  validating,
  publishing,
  forcePublish,
  gateBusy,
  fetching,
  saving,
  publishCheck,
  validateResult,
  publishBlocked,
  publishSuccess,
  publishCheckError,
  validateError,
  publishError,
  onForcePublishChange,
  onCheckPublish,
  onValidateChanges,
  onPublish,
}: CampaignEditorAdvancedPublishSectionProps) {
  return (
    <section className="flex flex-col gap-4" >
      <h2 className={adminTypography.sectionTitle}>Publish gate</h2>
      <div className="flex flex-wrap gap-2" >
        <Button
          type="button"
          variant="secondary"
          disabled={gateBusy || fetching}
          onClick={onCheckPublish}
        >
          {checking ? 'Checking...' : 'Check publish'}
        </Button>
        <Button
          type="button"
          variant="secondary"
          disabled={gateBusy || fetching || saving}
          onClick={onValidateChanges}
        >
          {validating ? 'Validating...' : 'Validate changes'}
        </Button>
        <Button type="button" disabled={gateBusy || fetching || saving} onClick={onPublish}>
          {publishing ? 'Publishing...' : 'Publish'}
        </Button>
      </div>

      <div className="flex items-center gap-2" >
        <Checkbox
          checked={forcePublish}
          disabled={gateBusy || fetching}
          id="campaign-force-publish"
          onCheckedChange={(checked) => onForcePublishChange(checked === true)}
        />
        <Label htmlFor="campaign-force-publish">Force publish</Label>
      </div>

      {publishCheckError
        ? editorApiErrorBlock(
            publishCheckError,
            'Publish check unavailable',
            'Could not check publish gate'
          )
        : null}
      {validateError
        ? editorApiErrorBlock(validateError, 'Validate unavailable', 'Could not validate changes')
        : null}
      {publishError
        ? editorApiErrorBlock(publishError, 'Publish unavailable', 'Could not publish campaign')
        : null}

      {publishSuccess ? <Badge variant="secondary">Campaign published</Badge> : null}

      {publishCheck ? (
        <div className={cn(campaignEditorSectionClass, 'gap-3')} >
          <div className="flex flex-wrap items-center gap-2" >
            <p className={adminTypography.label}>Publish check</p>
            <ValidityBadge valid={publishCheck.valid} validLabel="Ready" invalidLabel="Blocked" />
          </div>
          <FieldErrorsPanel title="Field errors" fieldErrors={publishCheck.field_errors} />
          <StringList title="Warnings" items={publishCheck.warning_slugs} />
        </div>
      ) : null}

      {validateResult ? (
        <div className={cn(campaignEditorSectionClass, 'gap-3')} >
          <div className="flex flex-wrap items-center gap-2" >
            <p className={adminTypography.label}>Patch validation</p>
            <ValidityBadge valid={validateResult.valid} validLabel="Valid" invalidLabel="Invalid" />
          </div>
          <FieldErrorsPanel title="Field errors" fieldErrors={validateResult.field_errors} />
          <StringList title="Warnings" items={validateResult.warnings} />
        </div>
      ) : null}

      {publishBlocked ? (
        <div className={cn(campaignEditorSectionClass, 'gap-3 border-destructive/50')} >
          <div className="flex flex-wrap items-center gap-2" >
            <p className={cn(adminTypography.label, "text-destructive")}>Publish blocked</p>
            <Badge variant="destructive">422</Badge>
          </div>
          <FieldErrorsPanel title="Field errors" fieldErrors={publishBlocked.field_errors} />
          <StringList title="Warning slugs" items={publishBlocked.warning_slugs} />
        </div>
      ) : null}
    </section>
  );
}
