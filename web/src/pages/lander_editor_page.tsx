import { useCallback, useEffect, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import { can } from '../helpers/permissions.js';
import { mapServiceError } from '../helpers/service_error.js';
import { pushToastMessage } from '../helpers/toast_ui.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import {
  fetchHostedEditorFile,
  fetchHostedEditorState,
  publishHostedLanderDraft,
  saveHostedEditorFile,
  type HostedEditorFileDTO,
  type HostedEditorStateDTO,
} from '../helpers/flows_api.js';
import { Breadcrumbs } from '../components/breadcrumbs.js';
import { Button } from '../components/button.js';
import { ErrorBlock } from '../components/error_block.js';

/**
 * Pick the default file to open in the editor from the hosted file list.
 * @param files - Files returned by the hosted editor state API.
 */
function defaultEditorFile(files: HostedEditorFileDTO[]): string {
  const editable = files.filter((f) => f.editable);
  if (!editable.length) return '';
  const index = editable.find((f) => f.path === 'index.html' || f.path.endsWith('/index.html'));
  return index?.path ?? editable[0]?.path ?? '';
}

export function LanderEditorPage() {
  const { id = '' } = useParams();
  const canWrite = can(auth.getUser()?.permissions ?? [], 'campaigns:write');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [busy, setBusy] = useState(false);
  const [state, setState] = useState<HostedEditorStateDTO | null>(null);
  const [selectedPath, setSelectedPath] = useState('');
  const [content, setContent] = useState('');
  const [dirty, setDirty] = useState(false);

  const reload = useCallback(async () => {
    if (!id) {
      setError(new Error('Missing lander id'));
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    const [editorState, err] = await to(fetchHostedEditorState(id));
    setLoading(false);
    if (err) {
      setError(err);
      return;
    }
    setState(editorState);
    setSelectedPath((current) => current || defaultEditorFile(editorState.files));
  }, [id]);

  const loadFile = useCallback(
    async (relPath: string) => {
      if (!id || !relPath) return;
      setBusy(true);
      const [body, err] = await to(fetchHostedEditorFile(id, relPath));
      setBusy(false);
      if (err) {
        setError(err);
        return;
      }
      setContent(body);
      setDirty(false);
    },
    [id]
  );

  useEffect(() => {
    void reload();
  }, [reload]);

  useEffect(() => {
    if (!selectedPath || loading || error) return;
    void loadFile(selectedPath);
  }, [selectedPath, loading, error, loadFile]);

  const onSave = async () => {
    if (!canWrite || busy || !id || !selectedPath) return;
    setBusy(true);
    const [result, err] = await to(saveHostedEditorFile(id, selectedPath, content));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Save failed', message: mapServiceError(err).message });
      return;
    }
    setDirty(false);
    setState((prev) =>
      prev
        ? {
            ...prev,
            draft_version: result.draft_version,
            has_unpublished_draft: result.has_unpublished_draft,
          }
        : prev
    );
    pushToastMessage({ title: 'Draft saved', message: selectedPath });
    await reload();
  };

  const onPublish = async () => {
    if (!canWrite || busy || !id || !state) return;
    setBusy(true);
    const [, err] = await to(publishHostedLanderDraft(id, state.draft_version));
    setBusy(false);
    if (err) {
      if (err instanceof ConfirmCancelledError) return;
      pushToastMessage({ title: 'Publish failed', message: mapServiceError(err).message });
      return;
    }
    pushToastMessage({ title: 'Lander published', message: state.name });
    await reload();
  };

  if (error) {
    return <ErrorBlock error={error} />;
  }

  const editableFiles = state?.files.filter((f) => f.editable) ?? [];

  return (
    <>
      <div className="page-header">
        <Breadcrumbs
          items={[
            { label: 'Campaigns', href: '/campaigns' },
            { label: 'Flows', href: '/campaigns/flows' },
            { label: state?.name ?? 'Lander editor' },
          ]}
        />
        <h1 className="page-header__title">{state?.name ?? 'Lander editor'}</h1>
        <p className="text-muted text-sm">
          Edit hosted lander files. Saves create a new draft version; publish promotes draft to live
          traffic.
        </p>
        {state ? (
          <p className="text-sm">
            Draft v{state.draft_version}
            {state.published_version > 0 ? `, published v${state.published_version}` : null}
            {state.has_unpublished_draft ? (
              <span className="text-warning">, unpublished changes</span>
            ) : null}
          </p>
        ) : null}
      </div>

      <div className="section-block stack">
        <div className="toolbar-row">
          {state?.preview_url ? (
            <a
              className="button button--sm button--secondary"
              href={state.preview_url}
              target="_blank"
              rel="noreferrer"
            >
              Open preview
            </a>
          ) : null}
          {canWrite && state?.has_unpublished_draft ? (
            <Button
              label={busy ? 'Publishing...' : 'Publish draft'}
              variant="primary"
              size="sm"
              loading={busy}
              disabled={busy}
              data-testid="lander-editor-publish"
              onClick={() => void onPublish()}
            />
          ) : null}
          {canWrite ? (
            <Button
              label={busy ? 'Saving...' : 'Save file'}
              variant="secondary"
              size="sm"
              loading={busy}
              disabled={busy || !selectedPath || !dirty}
              data-testid="lander-editor-save"
              onClick={() => void onSave()}
            />
          ) : null}
          <Link className="button button--sm button--ghost" to="/campaigns/flows">
            Back to flows
          </Link>
        </div>

        <div className="grid-2">
          <section className="section-card" data-testid="lander-editor-files">
            <h3 className="subsection-title">Files</h3>
            {loading ? <p className="text-muted text-sm">Loading...</p> : null}
            {!loading && editableFiles.length === 0 ? (
              <p className="text-muted text-sm">No editable text files in this draft.</p>
            ) : null}
            <ul className="stack">
              {editableFiles.map((file) => (
                <li key={file.path}>
                  <button
                    type="button"
                    className="button button--sm button--ghost"
                    aria-current={file.path === selectedPath ? 'true' : undefined}
                    onClick={() => {
                      if (file.path !== selectedPath) {
                        setSelectedPath(file.path);
                      }
                    }}
                  >
                    <span className="font-mono text-sm">{file.path}</span>
                    <span className="text-muted text-xs"> ({file.size} B)</span>
                  </button>
                </li>
              ))}
            </ul>
          </section>

          <section className="section-card stack" data-testid="lander-editor-pane">
            <h3 className="subsection-title">
              {selectedPath ? (
                <span className="font-mono text-sm">{selectedPath}</span>
              ) : (
                'Select a file'
              )}
            </h3>
            {canWrite && selectedPath ? (
              <textarea
                className="form-input"
                style={{ minHeight: '28rem', fontFamily: 'monospace' }}
                value={content}
                spellCheck={false}
                data-testid="lander-editor-textarea"
                onChange={(e) => {
                  setContent(e.target.value);
                  setDirty(true);
                }}
              />
            ) : selectedPath ? (
              <pre
                className="font-mono text-sm"
                style={{ minHeight: '28rem', whiteSpace: 'pre-wrap' }}
              >
                {content}
              </pre>
            ) : (
              <p className="text-muted text-sm">Choose a file from the list.</p>
            )}
          </section>
        </div>
      </div>
    </>
  );
}
