import { useEffect, useState } from 'react';
import {
  fetchHostedEditorState,
  fetchHostedFile,
  publishHostedLander,
  saveHostedFile,
  type HostedEditorState,
} from '../../helpers/landers_api.js';
import { ConfirmCancelledError } from '../../helpers/confirmed_api.js';
import { pushToastMessage } from '../../helpers/toast_ui.js';
import { ContextBar } from '../shell/context_bar.js';
import { PageChrome } from '../system/page_chrome.js';
import { Button } from '../system/button.js';
import { ErrorBlock } from '../system/error_block.js';
import { PageSkeleton } from '../system/page_skeleton.js';
import styles from './lander_editor_detail.module.css';

export type LanderEditorDetailProps = {
  landerId: string;
  onReload: () => void;
};

export function LanderEditorDetail({ landerId, onReload }: LanderEditorDetailProps) {
  const [state, setState] = useState<HostedEditorState | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<unknown>(null);
  const [selectedPath, setSelectedPath] = useState('');
  const [content, setContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [actionError, setActionError] = useState<unknown>(null);

  const loadState = () => {
    setLoading(true);
    setError(null);
    void fetchHostedEditorState(landerId)
      .then((data) => {
        setState(data);
        const firstEditable = data.files?.find((f) => f.editable !== false)?.path ?? data.files?.[0]?.path ?? '';
        if (firstEditable && !selectedPath) setSelectedPath(firstEditable);
      })
      .catch((err) => setError(err))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadState();
  }, [landerId]);

  useEffect(() => {
    if (!selectedPath) return;
    setFileLoading(true);
    setActionError(null);
    void fetchHostedFile(landerId, selectedPath)
      .then((text) => setContent(text))
      .catch((err) => setActionError(err))
      .finally(() => setFileLoading(false));
  }, [landerId, selectedPath]);

  const onSave = async () => {
    if (!selectedPath) return;
    setSaving(true);
    setActionError(null);
    try {
      await saveHostedFile(landerId, selectedPath, content);
      pushToastMessage({ title: 'Saved', message: selectedPath });
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setActionError(err);
    } finally {
      setSaving(false);
    }
  };

  const onPublish = async () => {
    setPublishing(true);
    setActionError(null);
    try {
      await publishHostedLander(landerId, state?.draft_version);
      pushToastMessage({ title: 'Published', message: 'Lander publish requested' });
      loadState();
      onReload();
    } catch (err) {
      if (err instanceof ConfirmCancelledError) return;
      setActionError(err);
    } finally {
      setPublishing(false);
    }
  };

  if (loading && !state) return <PageSkeleton rows={6} />;
  if (error) return <ErrorBlock error={error} fallbackTitle="Failed to load lander editor" />;

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <ContextBar parentLabel="Campaigns" parentTo="/campaigns" currentLabel={state?.name ?? landerId} />
        <PageChrome title={state?.name ?? 'Lander editor'} />
        {state?.has_unpublished_draft ? (
          <span className={styles.label}>Unpublished draft changes</span>
        ) : null}
      </div>
      <div className={styles.panel}>
        <span className={styles.label}>Files</span>
        <div className={styles.fileTree}>
          {(state?.files ?? []).map((file) => {
            const path = file.path ?? '';
            return (
              <button
                key={path}
                type="button"
                className={[styles.fileItem, path === selectedPath ? styles.fileItemActive : ''].join(' ')}
                onClick={() => setSelectedPath(path)}
              >
                {path || '-'}
              </button>
            );
          })}
        </div>
      </div>
      <div className={styles.editor}>
        {actionError ? <ErrorBlock error={actionError} fallbackTitle="Editor action failed" /> : null}
        <span className={styles.mono}>{selectedPath || 'Select a file'}</span>
        {fileLoading ? <PageSkeleton rows={4} /> : null}
        <textarea
          className={styles.textarea}
          value={content}
          disabled={!selectedPath || fileLoading}
          onChange={(e) => setContent(e.target.value)}
        />
        <div className={styles.actions}>
          <Button variant="primary" type="button" disabled={saving || !selectedPath} onClick={() => void onSave()}>
            {saving ? 'Saving...' : 'Save file'}
          </Button>
          <Button variant="secondary" type="button" disabled={publishing} onClick={() => void onPublish()}>
            {publishing ? 'Publishing...' : 'Publish'}
          </Button>
          {state?.preview_url ? (
            <a href={state.preview_url} target="_blank" rel="noreferrer">
              Preview
            </a>
          ) : null}
        </div>
      </div>
    </div>
  );
}
