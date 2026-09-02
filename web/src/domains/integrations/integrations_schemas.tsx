import { PageChrome } from '@/shell/page_chrome';
import { EmptyState } from '@/shell/empty_state';
import { ErrorBlock } from '@/shell/error_block';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
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
import { IntegrationsNav, integrationsPanelError } from '@/domains/integrations/integrations_nav';
import { JsonDashboardView } from '@/domains/dashboards/json_dashboard_view';
import { displayTimestamp } from '@/lib/display';

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

export function IntegrationsSchemas({
  tab,
  onTabChange,
  schemas,
  templates,
  fetching,
  error,
  hasSnapshot,
  createForm,
  applyForm,
  importForm,
  viewSchema,
}: IntegrationsSchemasProps) {
  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Schemas and templates">
        <IntegrationsNav />
        {integrationsPanelError(error, 'Could not load integration schemas')}
      </PageChrome>
    );
  }

  return (
    <PageChrome title="Schemas and templates">
      <IntegrationsNav />

      <div className="flex flex-wrap gap-2">
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
        <section className="grid gap-4">
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

          <div className="grid gap-2">
            <h2 className="text-base font-semibold">Schemas</h2>
            {schemas.length === 0 ? (
              <EmptyState title="No schemas" description="Integration schema catalog is empty." />
            ) : (
              <div className="ui-table-frame">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Kind</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Updated</TableHead>
                      <TableHead className="w-[5rem]">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {schemas.map((row) => (
                      <TableRow
                        key={row.id}
                        className="cursor-pointer"
                        onClick={() => applyForm.onPrefillFromSchema(row)}
                      >
                        <TableCell>{row.name}</TableCell>
                        <TableCell>{row.kind}</TableCell>
                        <TableCell>{row.version}</TableCell>
                        <TableCell>{displayTimestamp(row.updated_at)}</TableCell>
                        <TableCell>
                          <Button
                            type="button"
                            variant="outline"
                            onClick={(event) => {
                              event.stopPropagation();
                              viewSchema.onView(row);
                            }}
                          >
                            View
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>

          {viewSchema.fetching ? <PageSkeleton /> : null}
          {viewSchema.error ? (
            <ErrorBlock title="Could not load schema" message={viewSchema.error.message} />
          ) : null}
          {viewSchema.schema ? (
            <section className="grid gap-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-base font-semibold">
                  {viewSchema.schema.name ?? viewSchema.schema.id}
                </h2>
                <Button type="button" variant="ghost" onClick={viewSchema.onClose}>
                  Close
                </Button>
              </div>
              <pre className="ui-table-frame overflow-x-auto rounded-2xl p-4 text-xs">
                {JSON.stringify(viewSchema.schema.schema, null, 2)}
              </pre>
              <JsonDashboardView
                payload={viewSchema.schema as unknown as Record<string, unknown>}
              />
            </section>
          ) : null}
        </section>
      ) : null}

      {tab === 'templates' ? (
        <section className="grid gap-4">
          <IntegrationTemplateImportForm
            draftTemplateNames={importForm.draftTemplateNames}
            importing={importForm.importing}
            importError={importForm.importError}
            importSuccess={importForm.importSuccess}
            importedCount={importForm.importedCount}
            onDraftTemplateNamesChange={importForm.onDraftTemplateNamesChange}
            onImport={importForm.onImport}
          />

          <div className="grid gap-2">
            <h2 className="text-base font-semibold">Templates</h2>
            {templates.length === 0 ? (
              <EmptyState title="No templates" description="Integration template catalog is empty." />
            ) : (
              <div className="ui-table-frame">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Name</TableHead>
                      <TableHead>Kind</TableHead>
                      <TableHead>Category</TableHead>
                      <TableHead>Version</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {templates.map((row) => (
                      <TableRow key={`${row.name}-${row.file}`}>
                        <TableCell>{row.name}</TableCell>
                        <TableCell>{row.kind}</TableCell>
                        <TableCell>{row.category}</TableCell>
                        <TableCell>{row.version}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            )}
          </div>
        </section>
      ) : null}

      {error && hasSnapshot ? integrationsPanelError(error, 'Refresh failed') : null}
    </PageChrome>
  );
}
