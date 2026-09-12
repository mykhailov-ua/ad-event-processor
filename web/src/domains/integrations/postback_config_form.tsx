import { integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { DirectoryFilterForm, FilterPanel } from '@/shell/filter_panel';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { PasswordInput } from '@/components/ui/password_input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { PostbackDryRunResult } from '@/api/types';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import { ValidationErrorBlock } from '@/shell/validation_error_block';
import { adminTypography } from '@/lib/admin_kit';

const POSTBACK_PROVIDERS = [
  'webhook',
  'facebook',
  'google',
  'tiktok',
  'taboola',
  'outbrain',
  'microsoft_ads',
] as const;

export type PostbackConfigFormProps = {
  draftCampaignId: string;
  draftProvider: string;
  draftUrlTemplate: string;
  draftTargetEvent: string;
  draftApiToken: string;
  draftTestEventCode: string;
  saving: boolean;
  testing: boolean;
  saveError: Error | undefined;
  testError: Error | undefined;
  formValidationError?: AdminValidationError;
  saveSuccess: boolean;
  testResult: PostbackDryRunResult | undefined;
  onDraftCampaignIdChange: (value: string) => void;
  onDraftProviderChange: (value: string) => void;
  onDraftUrlTemplateChange: (value: string) => void;
  onDraftTargetEventChange: (value: string) => void;
  onDraftApiTokenChange: (value: string) => void;
  onDraftTestEventCodeChange: (value: string) => void;
  onSave: () => void;
  onTest: () => void;
};

export function PostbackConfigForm({
  draftCampaignId,
  draftProvider,
  draftUrlTemplate,
  draftTargetEvent,
  draftApiToken,
  draftTestEventCode,
  saving,
  testing,
  saveError,
  testError,
  formValidationError,
  saveSuccess,
  testResult,
  onDraftCampaignIdChange,
  onDraftProviderChange,
  onDraftUrlTemplateChange,
  onDraftTargetEventChange,
  onDraftApiTokenChange,
  onDraftTestEventCodeChange,
  onSave,
  onTest,
}: PostbackConfigFormProps) {
  const canSave =
    draftCampaignId.trim().length > 0 &&
    draftProvider.trim().length > 0 &&
    draftUrlTemplate.trim().length > 0;
  const canTest = draftCampaignId.trim().length > 0;

  const urlTemplateHint =
    draftProvider === 'google'
      ? 'Google: customer_id|conversion_action_id (e.g. 1234567890|987654321) or customers/123/conversionActions/456. Developer token goes in Test event code.'
      : draftProvider === 'microsoft_ads'
        ? 'Microsoft Ads: account_id|customer_id|conversion_name. Developer token in Test event code.'
        : null;

  return (
    <FilterPanel>
      <h2 className={adminTypography.sectionTitle}>Upsert postback config</h2>
      <p className={adminTypography.bodyMuted}>
        API token is encrypted at rest. Leave token empty on update to keep the existing value.
        Click a config row below to prefill this form.
      </p>

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="postback-campaign-id">Campaign ID</Label>
          <Input
            id="postback-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="postback-provider">Provider</Label>
          <Select value={draftProvider} onValueChange={onDraftProviderChange}>
            <SelectTrigger className="w-full" id="postback-provider">
              <SelectValue placeholder="Select provider" />
            </SelectTrigger>
            <SelectContent>
              {POSTBACK_PROVIDERS.map((provider) => (
                <SelectItem key={provider} value={provider}>
                  {provider}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-2">
          <Label htmlFor="postback-target-event">Target event</Label>
          <Input
            id="postback-target-event"
            value={draftTargetEvent}
            onChange={(event) => onDraftTargetEventChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="postback-url-template">URL template</Label>
          <Input
            id="postback-url-template"
            value={draftUrlTemplate}
            onChange={(event) => onDraftUrlTemplateChange(event.target.value)}
            placeholder={
              draftProvider === 'google'
                ? '1234567890|987654321'
                : draftProvider === 'microsoft_ads'
                  ? 'account_id|customer_id|conversion_name'
                  : undefined
            }
          />
          {urlTemplateHint ? (
            <p className={adminTypography.captionPlain}>{urlTemplateHint}</p>
          ) : null}
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="postback-api-token">API token</Label>
          <PasswordInput
            id="postback-api-token"
            autoComplete="off"
            value={draftApiToken}
            onChange={(event) => onDraftApiTokenChange(event.target.value)}
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="postback-test-event-code">Test event code</Label>
          <Input
            id="postback-test-event-code"
            value={draftTestEventCode}
            onChange={(event) => onDraftTestEventCodeChange(event.target.value)}
          />
        </div>
        <Button disabled={saving || !canSave} onClick={onSave} type="button">
          {saving ? 'Saving...' : 'Save config'}
        </Button>
        <Button disabled={testing || !canTest} onClick={onTest} type="button" variant="outline">
          {testing ? 'Testing...' : 'Dry-run test'}
        </Button>
      </DirectoryFilterForm>

      {formValidationError ? (
        <ValidationErrorBlock error={formValidationError} title="Check postback fields" />
      ) : null}
      {saveError ? integrationsPanelError(saveError, 'Save failed') : null}
      {testError ? integrationsPanelError(testError, 'Dry-run failed') : null}
      {saveSuccess ? <p>Config saved. List refreshed.</p> : null}
      {testResult ? (
        <div>
          <p>
            Dry-run {testResult.ok ? 'succeeded' : 'failed'} ({testResult.provider})
          </p>
          {testResult.http_status != null ? <p>HTTP status: {testResult.http_status}</p> : null}
          {testResult.error ? <p>{testResult.error}</p> : null}
          {testResult.rendered_url ? (
            <p className="text-destructive">{testResult.rendered_url}</p>
          ) : null}
        </div>
      ) : null}
    </FilterPanel>
  );
}
