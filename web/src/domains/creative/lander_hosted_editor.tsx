import { useMemo } from 'react';
import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { FilterField, FILTER_PANEL_NARROW_CLASS } from '@/shell/filter_panel';
import type { HostedEditorState } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { ActionLinksBand, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { adminKit, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { FileUploadField } from '@/shell/file_upload_field';
import type { DirectoryOverviewField } from '@/shell/directory_overview_dialog';
import {
  DirectorySelectOverviewTable,
  directoryOperateRows,
  directoryRecordMap,
} from '@/shell/directory_select_overview_table';
import { DirectoryRowActionsMenu } from '@/shell/directory_row_actions_menu';

export type LanderHostedEditorProps = {
  landerId: string;
  state: HostedEditorState | undefined;
  fetching: boolean;
  error: Error | undefined;
  hasSnapshot: boolean;
  acting?: boolean;
  actionError?: Error;
  actionMessage?: string;
  selectedFilePath?: string;
  fileContent?: string;
  fileLoading?: boolean;
  fileSaving?: boolean;
  fileError?: Error;
  onUploadZip?: (file: File) => void;
  onPublish?: () => void;
  onSelectFile?: (filePath: string) => void;
  onFileContentChange?: (value: string) => void;
  onSaveFile?: () => void;
};

function buildHostedFileOverviewFields(
  file: NonNullable<HostedEditorState['files']>[number]
): DirectoryOverviewField[] {
  return [
    { label: 'Path', value: file.path },
    { label: 'Size', value: file.size ?? '-' },
    { label: 'Editable', value: file.editable ? 'yes' : 'no' },
  ];
}

export function LanderHostedEditor({
  landerId,
  state,
  fetching,
  error,
  hasSnapshot,
  acting = false,
  actionError,
  actionMessage,
  selectedFilePath,
  fileContent = '',
  fileLoading = false,
  fileSaving = false,
  fileError,
  onUploadZip,
  onPublish,
  onSelectFile,
  onFileContentChange,
  onSaveFile,
}: LanderHostedEditorProps) {
  const fileByPath = useMemo(
    () => directoryRecordMap(state?.files, (file) => file.path),
    [state?.files]
  );
  const fileRows = useMemo(
    () => directoryOperateRows(state?.files, (file) => file.path, (file) => file.path),
    [state?.files]
  );

  if (fetching && !hasSnapshot && !error) {
    return <PageSkeleton />;
  }

  if (error && !hasSnapshot) {
    return (
      <PageChrome title="Hosted lander editor">
        <CreativeDirectoryStack>
          {creativePanelError(error, 'Could not load hosted editor')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  if (!state) {
    return (
      <PageChrome title="Hosted lander editor">
        <CreativeDirectoryStack>
          {creativePanelError(new Error('Editor state missing'), 'Could not load hosted editor')}
        </CreativeDirectoryStack>
      </PageChrome>
    );
  }

  const previewUrl = state.preview_url?.trim();

  return (
    <PageChrome title={`Hosted editor: ${state.name}`}>
      <CreativeDirectoryStack>
        <MetaLinksBand>
          <Link to="/landers">Back to landers</Link>
        </MetaLinksBand>

        <section className="grid gap-2">
          <h2 className={adminTypography.sectionTitle}>Draft status</h2>
          <ActionLinksBand className={adminTypography.body}>
            <Badge variant="outline">Draft v{state.draft_version}</Badge>
            <Badge variant="outline">Published v{state.published_version}</Badge>
            {state.has_unpublished_draft ? (
              <Badge variant="secondary">Unpublished draft</Badge>
            ) : null}
          </ActionLinksBand>
        </section>

        {onUploadZip || onPublish ? (
          <section className={FILTER_PANEL_NARROW_CLASS}>
            <h2 className={adminTypography.sectionTitle}>Hosted actions</h2>
            {onUploadZip ? (
              <FilterField htmlFor="lander-upload-zip" label="Upload ZIP">
                <FileUploadField
                  accept=".zip,application/zip"
                  buttonLabel="Choose ZIP"
                  disabled={acting}
                  emptyLabel="No ZIP selected"
                  id="lander-upload-zip"
                  onFileChange={(file) => {
                    if (file) {
                      onUploadZip(file);
                    }
                  }}
                />
              </FilterField>
            ) : null}
            {onPublish ? (
              <Button disabled={acting} onClick={onPublish} type="button">
                {acting ? 'Publishing...' : 'Publish hosted lander'}
              </Button>
            ) : null}
            {actionMessage ? (
              <p className={adminTypography.bodyMuted} role="status">
                {actionMessage}
              </p>
            ) : null}
            {actionError ? creativePanelError(actionError, 'Hosted lander action failed') : null}
          </section>
        ) : null}

        {previewUrl ? (
          <section className="grid gap-2">
            <h2 className={adminTypography.sectionTitle}>Preview</h2>
            <p className={adminTypography.body}>
              <a
                className="text-foreground underline"
                href={previewUrl}
                rel="noreferrer"
                target="_blank"
              >
                Open server preview
              </a>
            </p>
            <iframe
              className={cn(
                'min-h-[480px] w-full border border-border/50 bg-muted/40',
                adminKit.panelRadius
              )}
              src={previewUrl}
              title={`Preview for lander ${landerId}`}
            />
          </section>
        ) : null}

        <section className="grid gap-2">
          <h2 className={adminTypography.sectionTitle}>Files</h2>
          <TableHost>
            <DirectorySelectOverviewTable
              buildOverviewFields={buildHostedFileOverviewFields}
              disabled={acting}
              nameColumnLabel="Path"
              overviewTitle={(file) => file.path}
              recordById={fileByPath}
              renderActions={(row, _file, openOverview) => (
                <DirectoryRowActionsMenu
                  ariaLabel={`File actions ${row.id}`}
                  disabled={acting}
                  onOverview={openOverview}
                />
              )}
              rows={fileRows}
              selectedId={selectedFilePath ?? null}
              onSelectedIdChange={(path) => {
                if (path && onSelectFile) {
                  onSelectFile(path);
                }
              }}
            />
          </TableHost>
        </section>

        {selectedFilePath && onSaveFile ? (
          <section className="grid gap-4">
            <h2 className={adminTypography.sectionTitle}>Edit file</h2>
            <p className={cn(adminTypography.captionPlain, 'text-muted-foreground')}>{selectedFilePath}</p>
            {fileLoading ? (
              <p className={adminTypography.bodyMuted}>Loading file...</p>
            ) : (
              <>
                <FilterField htmlFor="lander-file-content" label="Content">
                  <Textarea
                    id="lander-file-content"
                    className="min-h-64"
                    value={fileContent}
                    onChange={(event) => onFileContentChange?.(event.target.value)}
                  />
                </FilterField>
                {fileError ? creativePanelError(fileError, 'Could not save file') : null}
                <div>
                  <PrimaryActionButton
                    disabled={fileLoading}
                    loading={fileSaving}
                    onClick={onSaveFile}
                    type="button"
                  >
                    Save file
                  </PrimaryActionButton>
                </div>
              </>
            )}
          </section>
        ) : null}

        {error && hasSnapshot ? creativePanelError(error, 'Refresh failed') : null}
      </CreativeDirectoryStack>
    </PageChrome>
  );
}
