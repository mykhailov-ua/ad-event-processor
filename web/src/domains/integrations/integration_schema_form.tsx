import { useEffect, useState } from 'react';

import { integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { DirectoryFilterForm, FilterField, FilterPanel } from '@/shell/filter_panel';
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
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create schema</DialogTitle>
          </DialogHeader>
          <p>Register a new integration schema definition. Schema body must be valid JSON.</p>
          <div>
            <div>
              <Label htmlFor="integration-schema-name">Name</Label>
              <Input
                id="integration-schema-name"
                value={draftName}
                onChange={(event) => onDraftNameChange(event.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="integration-schema-version">Version</Label>
              <Input
                id="integration-schema-version"
                type="number"
                min={1}
                value={draftVersion}
                onChange={(event) => onDraftVersionChange(event.target.value)}
              />
            </div>
            <div>
              <Label htmlFor="integration-schema-json">Schema JSON</Label>
              <Textarea
                id="integration-schema-json"
                value={draftSchemaJson}
                onChange={(event) => onDraftSchemaJsonChange(event.target.value)}
              />
            </div>
            {createError ? integrationsPanelError(createError, 'Create failed') : null}
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
    <FilterPanel>
      <h2>Apply schema</h2>
      <p>
        Apply a registered schema to a campaign. Click a schema row below to prefill the schema
        field.
      </p>

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="integration-apply-schema" label="Schema">
          {schemas.length > 0 ? (
            <Select value={draftSchemaId} onValueChange={onDraftSchemaIdChange}>
              <SelectTrigger id="integration-apply-schema">
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
        </FilterField>
        <FilterField htmlFor="integration-apply-campaign-id" label="Campaign ID">
          <Input
            id="integration-apply-campaign-id"
            value={draftCampaignId}
            onChange={(event) => onDraftCampaignIdChange(event.target.value)}
          />
        </FilterField>
        <Button disabled={applying || !canApply} onClick={onApply} type="button">
          {applying ? 'Applying...' : 'Apply schema'}
        </Button>
      </DirectoryFilterForm>

      {applyError ? integrationsPanelError(applyError, 'Apply failed') : null}
      {applySuccess ? <p>Schema applied to campaign.</p> : null}
      {applyResult ? (
        <div>
          <p>
            Status: {applyResult.status} ({applyResult.kind})
          </p>
          {applyResult.url_template ? <p>URL: {applyResult.url_template}</p> : null}
          {applyResult.panel_postback_url ? (
            <p>Postback: {applyResult.panel_postback_url}</p>
          ) : null}
          {applyResult.target_url ? <p>Target: {applyResult.target_url}</p> : null}
          {applyResult.mappings_applied_count != null ? (
            <p>Mappings applied: {applyResult.mappings_applied_count}</p>
          ) : null}
        </div>
      ) : null}
    </FilterPanel>
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
    <FilterPanel>
      <h2>Import templates</h2>
      <p>
        Import integration templates from the catalog into registered schemas. Leave names empty to
        import all templates. Use comma-separated names to import a subset.
      </p>

      <DirectoryFilterForm layout="auto-fill" onSubmit={(event) => event.preventDefault()}>
        <FilterField htmlFor="integration-import-names" label="Template names (optional)">
          <Input
            id="integration-import-names"
            value={draftTemplateNames}
            onChange={(event) => onDraftTemplateNamesChange(event.target.value)}
            placeholder="e.g. facebook_capi, tiktok_events"
          />
        </FilterField>
        <Button disabled={importing} onClick={onImport} type="button">
          {importing ? 'Importing...' : 'Import templates'}
        </Button>
      </DirectoryFilterForm>

      {importError ? integrationsPanelError(importError, 'Import failed') : null}
      {importSuccess ? (
        <p>
          Templates imported
          {importedCount != null
            ? ` (${importedCount} schema${importedCount === 1 ? '' : 's'})`
            : ''}
          . List refreshed.
        </p>
      ) : null}
    </FilterPanel>
  );
}
