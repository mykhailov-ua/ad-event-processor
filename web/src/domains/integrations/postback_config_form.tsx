import { ErrorBlock } from '@/components/system/error_block';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { PostbackDryRunResult } from '@/api/types';

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

  return (
    <section className="ui-filter-panel">
      <h2 className="text-base font-semibold">Upsert postback config</h2>
      <p className="text-sm text-muted-foreground">
        API token is encrypted at rest. Leave token empty on update to keep the existing value.
        Click a config row below to prefill this form.
      </p>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
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
            <SelectTrigger id="postback-provider" className="h-9 w-full text-sm">
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
          />
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="postback-api-token">API token</Label>
          <Input
            id="postback-api-token"
            type="password"
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
      </div>

      {saveError ? <ErrorBlock title="Save failed" message={saveError.message} /> : null}
      {testError ? <ErrorBlock title="Dry-run failed" message={testError.message} /> : null}
      {saveSuccess ? (
        <p className="text-sm text-muted-foreground">Config saved. List refreshed.</p>
      ) : null}
      {testResult ? (
        <div className="ui-surface grid gap-1 p-3 text-sm">
          <p>
            Dry-run {testResult.ok ? 'succeeded' : 'failed'} ({testResult.provider})
          </p>
          {testResult.http_status != null ? <p>HTTP status: {testResult.http_status}</p> : null}
          {testResult.error ? <p className="text-destructive">{testResult.error}</p> : null}
          {testResult.rendered_url ? (
            <p className="break-all font-mono text-xs">{testResult.rendered_url}</p>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}
