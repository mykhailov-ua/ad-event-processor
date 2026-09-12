import { useCallback, useEffect, useMemo, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import { isAbortError } from '@/api/client';
import {
  getHostedEditorState,
  publishHostedLander,
  readHostedEditorFile,
  saveHostedEditorFile,
  uploadHostedLanderZip,
} from '@/api/landers_hosted_api';
import { useResource } from '@/api/use_resource';
import { compileLanderBlocksToHtml } from '@/domains/creative/lander_block_compiler';
import type { LanderBlock } from '@/domains/creative/lander_block_model';
import { newLanderBlock } from '@/domains/creative/lander_block_model';
import { useSession } from '@/hooks/use_session';
import { useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { toError } from '@/lib/admin_error';
import { validationError } from '@/lib/admin_validation_error';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';

const INDEX_HTML = 'index.html';

export function useLanderHostedEditorPageWorkspace() {
  const { id: landerId } = useParams<{ id: string }>();
  const { user } = useSession();
  const canWrite = user?.permissions?.includes('campaigns:write') ?? false;
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const [blocks, setBlocks] = useState<LanderBlock[]>(() => [newLanderBlock('hero')]);
  const [blocksDirty, setBlocksDirty] = useState(false);
  const [blocksSaving, setBlocksSaving] = useState(false);
  const [blocksError, setBlocksError] = useState<Error | undefined>();

  const [selectedFilePath, setSelectedFilePath] = useState<string | undefined>();
  const [fileContent, setFileContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [fileSaving, setFileSaving] = useState(false);
  const [fileError, setFileError] = useState<Error | undefined>();

  const [acting, setActing] = useState(false);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [actionMessage, setActionMessage] = useState<string | undefined>();

  const { data, error, fetching, revalidating } = useResource(
    (signal) => {
      if (!landerId) {
        return Promise.reject(validationError('Lander id is required.', { field: 'id' }));
      }
      return getHostedEditorState(landerId, signal);
    },
    [landerId, refreshToken]
  );

  const hasSnapshot = data != null;
  useBreadcrumbSegmentLabel(landerId, data?.name);

  const blockPreviewHtml = useMemo(
    () => compileLanderBlocksToHtml(blocks, data?.name ?? 'Landing page'),
    [blocks, data?.name]
  );

  useEffect(() => {
    if (!selectedFilePath || !landerId) {
      return;
    }
    let cancelled = false;
    setFileLoading(true);
    setFileError(undefined);
    readHostedEditorFile(landerId, selectedFilePath)
      .then((body) => {
        if (!cancelled) {
          setFileContent(body.content);
        }
      })
      .catch((err: unknown) => {
        if (!cancelled && !isAbortError(err)) {
          setFileError(toError(err));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setFileLoading(false);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [landerId, selectedFilePath, refreshToken]);

  const onBlocksChange = useCallback((next: LanderBlock[]) => {
    setBlocks(next);
    setBlocksDirty(true);
  }, []);

  const onAddBlock = useCallback((type: LanderBlock['type']) => {
    setBlocks((current) => [...current, newLanderBlock(type)]);
    setBlocksDirty(true);
  }, []);

  const onSaveBlocks = useCallback(async () => {
    if (!landerId || !canWrite) {
      return;
    }
    setBlocksSaving(true);
    setBlocksError(undefined);
    try {
      const html = compileLanderBlocksToHtml(blocks, data?.name ?? 'Landing page');
      await saveHostedEditorFile(landerId, INDEX_HTML, html);
      setBlocksDirty(false);
      toast.success('Block layout saved to index.html');
      bumpRefresh();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setBlocksError(toError(err));
    } finally {
      setBlocksSaving(false);
    }
  }, [blocks, bumpRefresh, canWrite, data?.name, landerId]);

  const onUploadZip = useCallback(
    async (file: File) => {
      if (!landerId || !canWrite) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      setActionMessage(undefined);
      try {
        await uploadHostedLanderZip(landerId, file);
        toast.success('ZIP uploaded');
        bumpRefresh();
      } catch (err: unknown) {
        if (isAbortError(err)) {
          return;
        }
        setActionError(toError(err));
      } finally {
        setActing(false);
      }
    },
    [bumpRefresh, canWrite, landerId]
  );

  const onPublish = useCallback(async () => {
    if (!landerId || !canWrite || !data) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    try {
      await publishHostedLander(landerId, data.draft_version);
      setActionMessage('Published hosted lander');
      toast.success('Hosted lander published');
      bumpRefresh();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setActionError(toError(err));
    } finally {
      setActing(false);
    }
  }, [bumpRefresh, canWrite, data, landerId]);

  const onSaveFile = useCallback(async () => {
    if (!landerId || !canWrite || !selectedFilePath) {
      return;
    }
    setFileSaving(true);
    setFileError(undefined);
    try {
      await saveHostedEditorFile(landerId, selectedFilePath, fileContent);
      toast.success('File saved');
      bumpRefresh();
    } catch (err: unknown) {
      if (isAbortError(err)) {
        return;
      }
      setFileError(toError(err));
    } finally {
      setFileSaving(false);
    }
  }, [bumpRefresh, canWrite, fileContent, landerId, selectedFilePath]);

  return {
    landerId: landerId ?? '',
    canWrite,
    state: data,
    fetching: fetching || revalidating,
    error,
    hasSnapshot,
    acting,
    actionError,
    actionMessage,
    blocks,
    blocksDirty,
    blocksSaving,
    blocksError,
    blockPreviewHtml,
    onBlocksChange,
    onAddBlock,
    onSaveBlocks,
    selectedFilePath,
    onSelectFile: setSelectedFilePath,
    fileContent,
    onFileContentChange: setFileContent,
    fileLoading,
    fileSaving,
    fileError,
    onUploadZip: canWrite ? onUploadZip : undefined,
    onPublish: canWrite ? onPublish : undefined,
    onSaveFile: canWrite && selectedFilePath ? onSaveFile : undefined,
  };
}
