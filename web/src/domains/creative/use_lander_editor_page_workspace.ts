// L3 hosted lander editor: file tree state + upload/publish/save mutations; acting guard on refresh.
import { useCallback, useState } from 'react';
import { useParams } from 'react-router-dom';
import { toast } from 'sonner';

import {
  getHostedEditorState,
  getHostedLanderFile,
  publishHostedLander,
  putHostedLanderFile,
  uploadHostedLanderFiles,
} from '@/api/landers_api';
import { useCoalescedBumpRefresh, useRefreshToken } from '@/hooks/use_coalesced_refresh_token';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';
import { confirmDestructiveAction, mutationError } from '@/lib/mutation_audit';

export function useLanderEditorPageWorkspace() {
  const { id } = useParams();
  const landerId = id ?? '';
  const { refreshToken, bumpRefresh } = useRefreshToken();

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!landerId) {
        return Promise.reject(new Error('Lander ID required'));
      }
      return getHostedEditorState(landerId, signal);
    },
    [landerId, refreshToken]
  );

  const [acting, setActing] = useState(false);
  const bumpRefreshCoalesced = useCoalescedBumpRefresh(bumpRefresh, fetching || acting);
  const [actionError, setActionError] = useState<Error | undefined>();
  const [actionMessage, setActionMessage] = useState<string | undefined>();

  const [selectedFilePath, setSelectedFilePath] = useState<string | undefined>();
  const [fileContent, setFileContent] = useState('');
  const [fileLoading, setFileLoading] = useState(false);
  const [fileSaving, setFileSaving] = useState(false);
  const [fileError, setFileError] = useState<Error | undefined>();

  const onUploadZip = useCallback(
    (file: File) => {
      if (!landerId) {
        return;
      }
      setActing(true);
      setActionError(undefined);
      setActionMessage(undefined);
      void uploadHostedLanderFiles(landerId, file)
        .then(() => {
          setActionMessage('ZIP uploaded.');
          toast.success('ZIP uploaded');
          bumpRefreshCoalesced();
        })
        .catch((err: unknown) => {
          const nextError = mutationError(err);
          setActionError(nextError);
          toast.error(nextError.message);
        })
        .finally(() => {
          setActing(false);
        });
    },
    [bumpRefreshCoalesced, landerId]
  );

  const onPublish = useCallback(() => {
    if (!landerId) {
      return;
    }
    if (!confirmDestructiveAction('Publish hosted lander to production?')) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void publishHostedLander(landerId)
      .then(() => {
        setActionMessage('Publish accepted.');
        toast.success('Publish accepted');
        bumpRefreshCoalesced();
      })
      .catch((err: unknown) => {
        const nextError = mutationError(err);
        setActionError(nextError);
        toast.error(nextError.message);
      })
      .finally(() => {
        setActing(false);
      });
  }, [bumpRefreshCoalesced, landerId]);

  const onSelectFile = useCallback(
    (filePath: string) => {
      if (!landerId) {
        return;
      }
      setSelectedFilePath(filePath);
      setFileContent('');
      setFileError(undefined);
      setFileLoading(true);
      void getHostedLanderFile(landerId, filePath)
        .then((content) => {
          setFileContent(content);
        })
        .catch((err: unknown) => {
          const nextError = mutationError(err);
          setFileError(nextError);
        })
        .finally(() => {
          setFileLoading(false);
        });
    },
    [landerId]
  );

  const onSaveFile = useCallback(() => {
    if (!landerId || !selectedFilePath) {
      return;
    }
    setFileSaving(true);
    setFileError(undefined);
    void putHostedLanderFile(landerId, selectedFilePath, fileContent)
      .then(() => {
        toast.success('File saved');
        setActionMessage(`Saved ${selectedFilePath}.`);
        bumpRefreshCoalesced();
      })
      .catch((err: unknown) => {
        const nextError = mutationError(err);
        setFileError(nextError);
        toast.error(nextError.message);
      })
      .finally(() => {
        setFileSaving(false);
      });
  }, [bumpRefreshCoalesced, fileContent, landerId, selectedFilePath]);

  useBreadcrumbSegmentLabel(landerId || undefined, data?.name);

  return {
    landerId,
    state: data,
    fetching,
    error,
    hasSnapshot: data != null,
    acting,
    actionError,
    actionMessage,
    selectedFilePath,
    fileContent,
    fileLoading,
    fileSaving,
    fileError,
    onUploadZip,
    onPublish,
    onSelectFile,
    onFileContentChange: setFileContent,
    onSaveFile,
  };
}
