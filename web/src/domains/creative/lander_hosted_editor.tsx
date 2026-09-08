import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/shell/action_buttons';
import { PageChrome } from '@/shell/page_chrome';
import { PageSkeleton } from '@/shell/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Textarea } from '@/components/ui/textarea';
import { FilterField, FILTER_PANEL_NARROW_CLASS } from '@/shell/filter_panel';
import {
  DirectoryTable,
  DirectoryTableHead,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/shell/directory_table';
import type { HostedEditorState } from '@/api/types';
import { CreativeDirectoryStack } from '@/domains/creative/creative_directory_stack';
import { creativePanelError } from '@/domains/creative/creative_nav';
import { ActionLinksBand, MetaLinksBand, TableHost } from '@/shell/ui_bands';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

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
        <h2 className="text-base font-semibold">Draft status</h2>
        <ActionLinksBand className="text-sm">
          <Badge variant="outline">Draft v{state.draft_version}</Badge>
          <Badge variant="outline">Published v{state.published_version}</Badge>
          {state.has_unpublished_draft ? (
            <Badge variant="secondary">Unpublished draft</Badge>
          ) : null}
        </ActionLinksBand>
      </section>

      {onUploadZip || onPublish ? (
        <section className={FILTER_PANEL_NARROW_CLASS}>
          <h2 className="text-base font-semibold">Hosted actions</h2>
          {onUploadZip ? (
            <FilterField htmlFor="lander-upload-zip" label="Upload ZIP">
              <input
                id="lander-upload-zip"
                type="file"
                accept=".zip,application/zip"
                disabled={acting}
                onChange={(event) => {
                  const file = event.target.files?.[0];
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
            <p className="text-sm text-muted-foreground" role="status">
              {actionMessage}
            </p>
          ) : null}
          {actionError ? creativePanelError(actionError, 'Hosted lander action failed') : null}
        </section>
      ) : null}

      {previewUrl ? (
        <section className="grid gap-2">
          <h2 className="text-base font-semibold">Preview</h2>
          <p className="text-sm">
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
        <h2 className="text-base font-semibold">Files</h2>
        <TableHost>
          <DirectoryTable nested>
          <TableHeader>
            <TableRow>
              <DirectoryTableHead>Path</DirectoryTableHead>
              <DirectoryTableHead>Size</DirectoryTableHead>
              <DirectoryTableHead>Editable</DirectoryTableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {(state.files ?? []).map((file) => {
              const isSelected = selectedFilePath === file.path;
              const canEdit = file.editable && onSelectFile;
              return (
                <TableRow
                  key={file.path}
                  className={
                    canEdit
                      ? `cursor-pointer ${isSelected ? 'bg-muted/50' : 'hover:bg-muted/30'}`
                      : undefined
                  }
                  onClick={
                    canEdit
                      ? () => {
                          onSelectFile(file.path);
                        }
                      : undefined
                  }
                >
                  <TableCell className="font-mono text-xs">{file.path}</TableCell>
                  <TableCell>{file.size}</TableCell>
                  <TableCell>{file.editable ? 'yes' : 'no'}</TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </DirectoryTable>
        </TableHost>
      </section>

      {selectedFilePath && onSaveFile ? (
        <section className="grid gap-4">
          <h2 className="text-base font-semibold">Edit file</h2>
          <p className="font-mono text-xs text-muted-foreground">{selectedFilePath}</p>
          {fileLoading ? (
            <p className="text-sm text-muted-foreground">Loading file...</p>
          ) : (
            <>
              <FilterField htmlFor="lander-file-content" label="Content">
                <Textarea
                  id="lander-file-content"
                  className="min-h-64 font-mono text-sm"
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
