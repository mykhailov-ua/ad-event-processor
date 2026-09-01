import { Link } from 'react-router-dom';

import { PrimaryActionButton } from '@/components/system/action_buttons';
import { PageChrome } from '@/components/system/page_chrome';
import { PageSkeleton } from '@/components/system/page_skeleton';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import type { HostedEditorState } from '@/api/types';
import { CreativeNav, creativePanelError } from '@/domains/creative/creative_nav';

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
        <CreativeNav />
        {creativePanelError(error, 'Could not load hosted editor')}
      </PageChrome>
    );
  }

  if (!state) {
    return (
      <PageChrome title="Hosted lander editor">
        <CreativeNav />
        {creativePanelError(new Error('Editor state missing'), 'Could not load hosted editor')}
      </PageChrome>
    );
  }

  const previewUrl = state.preview_url?.trim();

  return (
    <PageChrome title={`Hosted editor: ${state.name}`}>
      <CreativeNav />
      <Link className="text-sm text-muted-foreground hover:underline" to="/landers">
        Back to landers
      </Link>

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Draft status</h2>
        <div className="flex flex-wrap gap-2 text-sm">
          <Badge variant="outline">Draft v{state.draft_version}</Badge>
          <Badge variant="outline">Published v{state.published_version}</Badge>
          {state.has_unpublished_draft ? (
            <Badge variant="secondary">Unpublished draft</Badge>
          ) : null}
        </div>
      </section>

      {onUploadZip || onPublish ? (
        <section className="ui-filter-panel gap-3">
          <h2 className="text-base font-semibold">Hosted actions</h2>
          {onUploadZip ? (
            <div className="grid gap-2">
              <Label htmlFor="lander-upload-zip">Upload ZIP</Label>
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
            </div>
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
            <a className="text-foreground underline" href={previewUrl} rel="noreferrer" target="_blank">
              Open server preview
            </a>
          </p>
          <iframe
            className="min-h-[480px] w-full rounded-xl border border-border/50 bg-muted/40"
            src={previewUrl}
            title={`Preview for lander ${landerId}`}
          />
        </section>
      ) : null}

      <section className="grid gap-2">
        <h2 className="text-base font-semibold">Files</h2>
        <div className="ui-table-frame">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Path</TableHead>
                <TableHead>Size</TableHead>
                <TableHead>Editable</TableHead>
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
          </Table>
        </div>
      </section>

      {selectedFilePath && onSaveFile ? (
        <section className="grid gap-4">
          <h2 className="text-base font-semibold">Edit file</h2>
          <p className="font-mono text-xs text-muted-foreground">{selectedFilePath}</p>
          {fileLoading ? (
            <p className="text-sm text-muted-foreground">Loading file…</p>
          ) : (
            <>
              <div className="grid gap-2">
                <Label htmlFor="lander-file-content">Content</Label>
                <Textarea
                  id="lander-file-content"
                  className="min-h-64 font-mono text-sm"
                  value={fileContent}
                  onChange={(event) => onFileContentChange?.(event.target.value)}
                />
              </div>
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
    </PageChrome>
  );
}
