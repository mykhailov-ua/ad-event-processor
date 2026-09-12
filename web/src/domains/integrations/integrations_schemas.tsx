import { useMemo, useState } from 'react';

import { EmptyState } from '@/shell/empty_state';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import type {
  ApplyIntegrationSchemaResponse,
  IntegrationSchema,
  IntegrationTemplateCatalogEntry,
} from '@/api/types';
import {
  IntegrationSchemaApplyForm,
  IntegrationSchemaCreateForm,
  IntegrationTemplateImportForm,
} from '@/domains/integrations/integration_schema_form';
import {
  IntegrationsPageWithLoad,
  integrationsPanelError,
} from '@/domains/integrations/integrations_nav';
import { adminSpacing, customerDetailSectionClass } from '@/lib/admin_spacing';
import { adminTypography } from '@/lib/admin_kit';
import { shellChrome } from '@/shell/shell_chrome';
import { cn } from '@/lib/utils';
import { ErrorBlock } from '@/shell/error_block';
import { displayTimestamp } from '@/lib/display';
import type { AdminValidationError } from '@/lib/admin_validation_error';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryRecordMap,
  directoryOperateRows,
} from '@/shell/directory_select_overview_table';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { JsonPayloadView } from '@/shell/json_payload_view';
import { TableHost } from '@/shell/ui_bands';

export type IntegrationsSchemasTab = 'schemas' | 'templates';

const SCHEMAS_TABS: { id: IntegrationsSchemasTab; label: string }[] = [
  { id: 'schemas', label: 'Schemas' },
  { id: 'templates', label: 'Templates' },
];

export type IntegrationsSchemasProps = {
  tab: IntegrationsSchemasTab;
  onTabChange: (tab: IntegrationsSchemasTab) => void;
  schemas: IntegrationSchema[];
  templates: IntegrationTemplateCatalogEntry[];
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  formValidationError?: AdminValidationError;
  createForm: {
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
  applyForm: {
    draftSchemaId: string;
    draftCampaignId: string;
    applying: boolean;
    applyError: Error | undefined;
    applySuccess: boolean;
    applyResult: ApplyIntegrationSchemaResponse | undefined;
    onDraftSchemaIdChange: (value: string) => void;
    onDraftCampaignIdChange: (value: string) => void;
    onApply: () => void;
    onPrefillFromSchema: (row: IntegrationSchema) => void;
  };
  importForm: {
    draftTemplateNames: string;
    importing: boolean;
    importError: Error | undefined;
    importSuccess: boolean;
    importedCount: number | undefined;
    onDraftTemplateNamesChange: (value: string) => void;
    onImport: () => void;
  };
  viewSchema: {
    schema: IntegrationSchema | undefined;
    fetching: boolean;
    error: Error | undefined;
    onView: (row: IntegrationSchema) => void;
    onClose: () => void;
  };
};

function templateId(row: IntegrationTemplateCatalogEntry): string {
  return `${row.name}-${row.file}`;
}

function buildSchemaOverviewFields(row: IntegrationSchema): DirectoryOverviewField[] {
  return [
    { label: 'Kind', value: row.kind },
    { label: 'Version', value: row.version },
    { label: 'Updated', value: displayTimestamp(row.updated_at) },
    { label: 'Created', value: displayTimestamp(row.created_at) },
    { label: 'ID', value: row.id },
  ];
}

function buildTemplateOverviewFields(
  row: IntegrationTemplateCatalogEntry
): DirectoryOverviewField[] {
  return [
    { label: 'Kind', value: row.kind },
    { label: 'Category', value: row.category },
    { label: 'Version', value: row.version },
    { label: 'File', value: row.file },
  ];
}

export function IntegrationsSchemas({
  tab,
  onTabChange,
  schemas,
  templates,
  fetching,
  error,
  hasSnapshot,
  formValidationError,
  createForm,
  applyForm,
  importForm,
  viewSchema,
}: IntegrationsSchemasProps) {
  const [selectedSchemaId, setSelectedSchemaId] = useState<string | null>(null);
  const [selectedTemplateId, setSelectedTemplateId] = useState<string | null>(null);

  const schemaRecordById = useMemo(() => directoryRecordMap(schemas, (row) => row.id), [schemas]);
  const schemaRows = useMemo(
    () =>
      directoryOperateRows(
        schemas,
        (row) => row.id,
        (row) => row.name ?? row.id
      ),
    [schemas]
  );

  const templateRecordById = useMemo(() => directoryRecordMap(templates, templateId), [templates]);
  const templateRows = useMemo(
    () => directoryOperateRows(templates, templateId, (row) => row.name ?? templateId(row)),
    [templates]
  );

  const handleSchemaSelectedIdChange = (id: string | null) => {
    setSelectedSchemaId(id);
    if (id) {
      const record = schemaRecordById.get(id);
      if (record) {
        applyForm.onPrefillFromSchema(record);
      }
    }
  };

  return (
    <IntegrationsPageWithLoad
      blockingErrorTitle="Could not load integration schemas"
      fetchState={{ error, fetching, hasSnapshot }}
      title="Schemas and templates"
    >
      {formValidationError ? (
        <ErrorBlock error={formValidationError} title="Check schema fields" />
      ) : null}
      <div className={adminSpacing.flex.buttonGroup}>
        {SCHEMAS_TABS.map((item) => (
          <Button
            key={item.id}
            type="button"
            variant={tab === item.id ? 'default' : 'outline'}
            onClick={() => onTabChange(item.id)}
          >
            {item.label}
          </Button>
        ))}
      </div>

      {tab === 'schemas' ? (
        <section className={customerDetailSectionClass}>
          <IntegrationSchemaCreateForm
            draftName={createForm.draftName}
            draftVersion={createForm.draftVersion}
            draftSchemaJson={createForm.draftSchemaJson}
            creating={createForm.creating}
            createError={createForm.createError}
            createSuccess={createForm.createSuccess}
            onDraftNameChange={createForm.onDraftNameChange}
            onDraftVersionChange={createForm.onDraftVersionChange}
            onDraftSchemaJsonChange={createForm.onDraftSchemaJsonChange}
            onCreate={createForm.onCreate}
          />

          <IntegrationSchemaApplyForm
            schemas={schemas}
            draftSchemaId={applyForm.draftSchemaId}
            draftCampaignId={applyForm.draftCampaignId}
            applying={applyForm.applying}
            applyError={applyForm.applyError}
            applySuccess={applyForm.applySuccess}
            applyResult={applyForm.applyResult}
            onDraftSchemaIdChange={applyForm.onDraftSchemaIdChange}
            onDraftCampaignIdChange={applyForm.onDraftCampaignIdChange}
            onApply={applyForm.onApply}
          />

          <div className={`grid ${adminSpacing.gap.md}`}>
            <h2 className={adminTypography.sectionTitle}>Schemas</h2>
            {schemas.length === 0 ? (
              <EmptyState title="No schemas" description="Integration schema catalog is empty." />
            ) : (
              <TableHost>
                <DirectorySelectOverviewTable
                  buildOverviewFields={buildSchemaOverviewFields}
                  disabled={fetching}
                  overviewTitle={(row) => row.name ?? row.id}
                  recordById={schemaRecordById}
                  renderActions={(tableRow, record, openOverview) => (
                    <DirectoryRowActionsMenu
                      ariaLabel={`Actions for ${String(tableRow.label)}`}
                      disabled={fetching}
                      onOverview={openOverview}
                    >
                      <DropdownMenuItem onClick={() => applyForm.onPrefillFromSchema(record)}>
                        Prefill apply form
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={() => viewSchema.onView(record)}>
                        View JSON
                      </DropdownMenuItem>
                    </DirectoryRowActionsMenu>
                  )}
                  rows={schemaRows}
                  selectedId={selectedSchemaId}
                  onSelectedIdChange={handleSchemaSelectedIdChange}
                />
              </TableHost>
            )}
          </div>

          {viewSchema.fetching ? <PageSkeleton /> : null}
          {viewSchema.error
            ? integrationsPanelError(viewSchema.error, 'Could not load schema')
            : null}
          {viewSchema.schema ? (
            <section className={shellChrome.sectionPanelClass}>
              <div className={cn('grid grid-cols-[1fr_auto] items-center', adminSpacing.gap.md)}>
                <h2 className={adminTypography.sectionTitle}>
                  {viewSchema.schema.name ?? viewSchema.schema.id}
                </h2>
                <Button type="button" variant="ghost" onClick={viewSchema.onClose}>
                  Close
                </Button>
              </div>
              <pre className={cn('overflow-x-auto bg-muted p-2', adminTypography.monoData)}>
                {JSON.stringify(viewSchema.schema.schema, null, 2)}
              </pre>
              <JsonPayloadView payload={viewSchema.schema} />
            </section>
          ) : null}
        </section>
      ) : null}

      {tab === 'templates' ? (
        <section className={customerDetailSectionClass}>
          <IntegrationTemplateImportForm
            draftTemplateNames={importForm.draftTemplateNames}
            importing={importForm.importing}
            importError={importForm.importError}
            importSuccess={importForm.importSuccess}
            importedCount={importForm.importedCount}
            onDraftTemplateNamesChange={importForm.onDraftTemplateNamesChange}
            onImport={importForm.onImport}
          />

          <div className={`grid ${adminSpacing.gap.md}`}>
            <h2 className={adminTypography.sectionTitle}>Templates</h2>
            {templates.length === 0 ? (
              <EmptyState
                title="No templates"
                description="Integration template catalog is empty."
              />
            ) : (
              <TableHost>
                <DirectorySelectOverviewTable
                  buildOverviewFields={buildTemplateOverviewFields}
                  disabled={fetching}
                  overviewTitle={(row) => row.name ?? templateId(row)}
                  recordById={templateRecordById}
                  rows={templateRows}
                  selectedId={selectedTemplateId}
                  onSelectedIdChange={setSelectedTemplateId}
                />
              </TableHost>
            )}
          </div>
        </section>
      ) : null}
    </IntegrationsPageWithLoad>
  );
}
