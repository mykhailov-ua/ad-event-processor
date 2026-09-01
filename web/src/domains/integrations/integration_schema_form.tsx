import { useEffect, useState } from 'react';

import { ErrorBlock } from '@/components/system/error_block';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import type { ApplyIntegrationSchemaResponse, IntegrationSchema } from '@/api/types';

export type IntegrationSchemaCreateFormProps = {
  draftName: string;
  draftVersion: string;
  draftSchemaJson: string;
  creating: boolean;
  createError: Error | undefined;
  createSuccess: boolean;
  onDraftNameChange: (value: string) => void;
  onDraftVersionChange: (value: string) => void;
  onDraftSchemaJsonChange: (value: string) => void;
  onCreate: () => void;
};

export function IntegrationSchemaCreateForm({
  draftName,
  draftVersion,
  draftSchemaJson,
  creating,
  createError,
  createSuccess,
  onDraftNameChange,
  onDraftVersionChange,
  onDraftSchemaJsonChange,
  onCreate,
}: IntegrationSchemaCreateFormProps) {
  const [createOpen, setCreateOpen] = useState(false);

  useEffect(() => {
    if (createSuccess) {
      setCreateOpen(false);
    }
  }, [createSuccess]);

  const canCreate =
    draftName.trim().length > 0 &&
    draftVersion.trim().length > 0 &&
    draftSchemaJson.trim().length > 0;

  return (
    <>
      <Button onClick={() => setCreateOpen(true)} type="button">
        Create schema
      </Button>
      <Dialog onOpenChange={setCreateOpen} open={createOpen}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Create schema</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Register a new integration schema definition. Schema body must be valid JSON.
          </p>
          <div className="grid gap-4">
            <div className="grid gap-2">
              <Label htmlFor="integration-schema-name">Name</Label>
              <Input
                id="integration-schema-name"
                value={draftName}
                onChange={(event) => onDraftNameChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="integration-schema-version">Version</Label>
              <Input
                id="integration-schema-version"
                type="number"
                min={1}
                value={draftVersion}
                onChange={(event) => onDraftVersionChange(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="integration-schema-json">Schema JSON</Label>
              <Textarea
                id="integration-schema-json"
                className="min-h-32 font-mono text-xs"
                value={draftSchemaJson}
                onChange={(event) => onDraftSchemaJsonChange(event.target.value)}
              />
            </div>
            {createError ? <ErrorBlock title="Create failed" message={createError.message} /> : null}
          </div>
          <DialogFooter>
            <Button disabled={creating || !canCreate} onClick={onCreate} type="button">
              {creating ? 'Creating...' : 'Create schema'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

export type IntegrationSchemaApplyFormProps = {
  schemas: IntegrationSchema[];
  draftSchemaId: string;
  draftCampaignId: string;
  applying: boolean;
  applyError: Error | undefined;
  applySuccess: boolean;
  applyResult: ApplyIntegrationSchemaResponse | undefined;
  onDraftSchemaIdChange: (value: string) => void;
  onDraftCampaignIdChange: (value: string) => void;
  onApply: () => void;
};

export function IntegrationSchemaApplyForm({
  schemas,
  draftSchemaId,
  draftCampaignId,
  applying,
  applyError,
  applySuccess,
  applyResult,
  onDraftSchemaIdChange,
  onDraftCampaignIdChange,
  onApply,
}: IntegrationSchemaApplyFormProps) {
  const canApply = draftSchemaId.trim().length > 0 && draftCampaignId.trim().length > 0;

  return (
    <section className="ui-filter-panel">
      <h2 className="text-base font-semibold">Apply schema</h2>
      <p className="text-sm text-muted-foreground">
        Apply a registered schema to a campaign. Click a schema row below to prefill the schema
        field.
      </p>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="integration-apply-schema">Schema</Label>
          {schemas.length > 0 ? (
            <Select value={draftSchemaId} onValueChange={onDraftSchemaIdChange}>
              <SelectTrigger id="integration-apply-schema" className="h-9 w-full text-sm">
                <SelectValue placeholder="Select schema" />
              </SelectTrigger>
              <SelectContent>
                {schemas.map((row) => (
                  <SelectItem key={row.id} value={row.id}>
                    {row.name} v{row.version} ({row.kind})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          ) : (
            <Input
              id="integration-apply-schema"
              value={draftSchemaId}
              onChange={(event) => onDraftSchemaIdChange(event.target.value)}
              placeholder="Schema UUID"
            />
          )}
        </div>
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="integration-apply-campaign-id">Campaign ID</Label>
          <Input
            id="integration-apply-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </div>
        <Button disabled={applying || !canApply} onClick={onApply} type="button">
          {applying ? 'Applying...' : 'Apply schema'}
        </Button>
      </div>

      {applyError ? <ErrorBlock title="Apply failed" message={applyError.message} /> : null}
      {applySuccess ? (
        <p className="text-sm text-muted-foreground">Schema applied to campaign.</p>
      ) : null}
      {applyResult ? (
        <div className="ui-surface grid gap-1 p-3 text-sm">
          <p>
            Status: {applyResult.status} ({applyResult.kind})
          </p>
          {applyResult.url_template ? (
            <p className="break-all font-mono text-xs">URL: {applyResult.url_template}</p>
          ) : null}
          {applyResult.panel_postback_url ? (
            <p className="break-all font-mono text-xs">
              Postback: {applyResult.panel_postback_url}
            </p>
          ) : null}
          {applyResult.target_url ? (
            <p className="break-all font-mono text-xs">Target: {applyResult.target_url}</p>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

export type IntegrationTemplateImportFormProps = {
  draftTemplateNames: string;
  importing: boolean;
  importError: Error | undefined;
  importSuccess: boolean;
  importedCount: number | undefined;
  onDraftTemplateNamesChange: (value: string) => void;
  onImport: () => void;
};

export function IntegrationTemplateImportForm({
  draftTemplateNames,
  importing,
  importError,
  importSuccess,
  importedCount,
  onDraftTemplateNamesChange,
  onImport,
}: IntegrationTemplateImportFormProps) {
  return (
    <section className="ui-filter-panel">
      <h2 className="text-base font-semibold">Import templates</h2>
      <p className="text-sm text-muted-foreground">
        Import integration templates from the catalog into registered schemas. Leave names empty to
        import all templates. Use comma-separated names to import a subset.
      </p>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(12rem,1fr))] items-end gap-4">
        <div className="grid gap-2 md:col-span-2">
          <Label htmlFor="integration-import-names">Template names (optional)</Label>
          <Input
            id="integration-import-names"
            value={draftTemplateNames}
            onChange={(event) => onDraftTemplateNamesChange(event.target.value)}
            placeholder="e.g. facebook_capi, tiktok_events"
          />
        </div>
        <Button disabled={importing} onClick={onImport} type="button">
          {importing ? 'Importing...' : 'Import templates'}
        </Button>
      </div>

      {importError ? <ErrorBlock title="Import failed" message={importError.message} /> : null}
      {importSuccess ? (
        <p className="text-sm text-muted-foreground">
          Templates imported
          {importedCount != null ? ` (${importedCount} schema${importedCount === 1 ? '' : 's'})` : ''}
          . List refreshed.
        </p>
      ) : null}
    </section>
  );
}
