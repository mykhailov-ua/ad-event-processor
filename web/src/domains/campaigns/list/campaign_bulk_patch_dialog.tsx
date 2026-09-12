import { Checkbox } from '@/components/ui/checkbox';
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
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { useCampaignBulkPatchDialogWorkspace } from '@/domains/campaigns/list/use_campaign_bulk_patch_dialog_workspace';
import { campaignPanelError } from '@/domains/campaigns/editor/campaign_editor_shared';
import { PrimaryActionButton, SecondaryActionButton } from '@/shell/action_buttons';
import { adminTypography } from '@/lib/admin_kit';
import { actionGuardError } from '@/lib/admin_validation_error';
import { ValidationErrorBlock } from '@/shell/validation_error_block';
import { cn } from '@/lib/utils';

const PACING_OPTIONS = [
  { value: 'ASAP', label: 'ASAP' },
  { value: 'EVEN', label: 'Even' },
  { value: 'VPP', label: 'VPP' },
];

const STATUS_OPTIONS = [
  { value: 'active', label: 'Active' },
  { value: 'paused', label: 'Paused' },
];

const FAILOVER_OPTIONS = [
  { value: 'reject', label: 'Reject (402)' },
  { value: 'fallback_url', label: 'Fallback URL' },
];

export type CampaignBulkPatchDialogProps = {
  campaignIds: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPatched?: () => void;
};

export function CampaignBulkPatchDialog({
  campaignIds,
  open,
  onOpenChange,
  onPatched,
}: CampaignBulkPatchDialogProps) {
  const {
    draft,
    enabled,
    onDraftChange,
    onEnabledChange,
    patching,
    patchError,
    validationError,
    results,
    onApplyPatch,
  } = useCampaignBulkPatchDialogWorkspace({
    campaignIds,
    open,
    onPatched,
  });

  const successCount = results?.filter((row) => row.ok).length ?? 0;
  const failedCount = results?.filter((row) => !row.ok).length ?? 0;
  const fieldsLocked = patching || successCount > 0;

  return (
    <Dialog onOpenChange={onOpenChange} open={open}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Bulk edit campaigns</DialogTitle>
          <DialogDescription>
            Apply the same field changes to {campaignIds.length} selected campaign(s). Only checked
            fields are sent to the server.
          </DialogDescription>
        </DialogHeader>
        <DialogBody className="grid gap-4">
          <BulkPatchTextField
            description="Offer or landing redirect URL"
            disabled={fieldsLocked}
            enabled={enabled.target_url}
            id="bulk-patch-target-url"
            label="Target URL"
            value={draft.target_url}
            onEnabledChange={(checked) => onEnabledChange('target_url', checked)}
            onValueChange={(value) => onDraftChange('target_url', value)}
          />
          <BulkPatchTextField
            description="Campaign budget limit (same format as campaign editor)"
            disabled={fieldsLocked}
            enabled={enabled.budget_limit}
            id="bulk-patch-budget-limit"
            label="Budget limit"
            value={draft.budget_limit}
            onEnabledChange={(checked) => onEnabledChange('budget_limit', checked)}
            onValueChange={(value) => onDraftChange('budget_limit', value)}
          />
          <BulkPatchSelectField
            disabled={fieldsLocked}
            enabled={enabled.pacing_mode}
            id="bulk-patch-pacing"
            label="Pacing mode"
            options={PACING_OPTIONS}
            value={draft.pacing_mode}
            onEnabledChange={(checked) => onEnabledChange('pacing_mode', checked)}
            onValueChange={(value) => onDraftChange('pacing_mode', value)}
          />
          <BulkPatchTextField
            description="Comma-separated ISO country codes (e.g. US, DE)"
            disabled={fieldsLocked}
            enabled={enabled.target_countries}
            id="bulk-patch-countries"
            label="Target countries"
            value={draft.target_countries}
            onEnabledChange={(checked) => onEnabledChange('target_countries', checked)}
            onValueChange={(value) => onDraftChange('target_countries', value)}
          />
          <BulkPatchSelectField
            disabled={fieldsLocked}
            enabled={enabled.status}
            id="bulk-patch-status"
            label="Status"
            options={STATUS_OPTIONS}
            value={draft.status}
            onEnabledChange={(checked) => onEnabledChange('status', checked)}
            onValueChange={(value) => onDraftChange('status', value)}
          />
          <BulkPatchTextField
            disabled={fieldsLocked}
            enabled={enabled.timezone}
            id="bulk-patch-timezone"
            label="Timezone"
            placeholder="UTC"
            value={draft.timezone}
            onEnabledChange={(checked) => onEnabledChange('timezone', checked)}
            onValueChange={(value) => onDraftChange('timezone', value)}
          />
          <BulkPatchTextField
            disabled={fieldsLocked}
            enabled={enabled.referrer_filter}
            id="bulk-patch-referrer"
            label="Referrer filter"
            value={draft.referrer_filter}
            onEnabledChange={(checked) => onEnabledChange('referrer_filter', checked)}
            onValueChange={(value) => onDraftChange('referrer_filter', value)}
          />
          <BulkPatchTextField
            disabled={fieldsLocked}
            enabled={enabled.fallback_click_url}
            id="bulk-patch-fallback-url"
            label="Fallback click URL"
            value={draft.fallback_click_url}
            onEnabledChange={(checked) => onEnabledChange('fallback_click_url', checked)}
            onValueChange={(value) => onDraftChange('fallback_click_url', value)}
          />
          <BulkPatchSelectField
            disabled={fieldsLocked}
            enabled={enabled.budget_failover_mode}
            id="bulk-patch-failover"
            label="Budget failover mode"
            options={FAILOVER_OPTIONS}
            value={draft.budget_failover_mode}
            onEnabledChange={(checked) => onEnabledChange('budget_failover_mode', checked)}
            onValueChange={(value) => onDraftChange('budget_failover_mode', value)}
          />

          {validationError ? (
            <ValidationErrorBlock
              error={actionGuardError(validationError)}
              title="Bulk edit validation"
            />
          ) : null}
          {patchError ? campaignPanelError(patchError, 'Bulk edit failed') : null}
          {results && failedCount > 0 ? (
            <p className={cn('m-0', adminTypography.captionPlain)}>
              {successCount} succeeded, {failedCount} failed. Close to review the list.
            </p>
          ) : null}
        </DialogBody>
        <DialogFooter>
          <SecondaryActionButton
            disabled={patching}
            type="button"
            onClick={() => onOpenChange(false)}
          >
            {successCount > 0 ? 'Close' : 'Cancel'}
          </SecondaryActionButton>
          <PrimaryActionButton
            disabled={patching || successCount > 0}
            loading={patching}
            type="button"
            onClick={() => void onApplyPatch()}
          >
            Apply to {campaignIds.length} campaign(s)
          </PrimaryActionButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

type BulkPatchTextFieldProps = {
  id: string;
  label: string;
  description?: string;
  placeholder?: string;
  value: string;
  enabled: boolean;
  disabled?: boolean;
  onValueChange: (value: string) => void;
  onEnabledChange: (checked: boolean) => void;
};

function BulkPatchTextField({
  id,
  label,
  description,
  placeholder,
  value,
  enabled,
  disabled,
  onValueChange,
  onEnabledChange,
}: BulkPatchTextFieldProps) {
  const inputId = `${id}-input`;
  return (
    <div className="grid gap-2">
      <div className="flex items-center gap-2">
        <Checkbox
          checked={enabled}
          disabled={disabled}
          id={id}
          onCheckedChange={(checked) => onEnabledChange(checked === true)}
        />
        <Label htmlFor={inputId}>{label}</Label>
      </div>
      {description ? (
        <p className={cn('m-0', adminTypography.captionPlain)}>{description}</p>
      ) : null}
      <Input
        disabled={disabled || !enabled}
        id={inputId}
        placeholder={placeholder}
        value={value}
        onChange={(event) => onValueChange(event.target.value)}
      />
    </div>
  );
}

type BulkPatchSelectFieldProps = {
  id: string;
  label: string;
  value: string;
  enabled: boolean;
  disabled?: boolean;
  options: { value: string; label: string }[];
  onValueChange: (value: string) => void;
  onEnabledChange: (checked: boolean) => void;
};

function BulkPatchSelectField({
  id,
  label,
  value,
  enabled,
  disabled,
  options,
  onValueChange,
  onEnabledChange,
}: BulkPatchSelectFieldProps) {
  const selectId = `${id}-select`;
  return (
    <div className="grid gap-2">
      <div className="flex items-center gap-2">
        <Checkbox
          checked={enabled}
          disabled={disabled}
          id={id}
          onCheckedChange={(checked) => onEnabledChange(checked === true)}
        />
        <Label htmlFor={selectId}>{label}</Label>
      </div>
      <Select
        disabled={disabled || !enabled}
        value={value || undefined}
        onValueChange={onValueChange}
      >
        <SelectTrigger id={selectId}>
          <SelectValue placeholder="Select value" />
        </SelectTrigger>
        <SelectContent>
          {options.map((option) => (
            <SelectItem key={option.value} value={option.value}>
              {option.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
