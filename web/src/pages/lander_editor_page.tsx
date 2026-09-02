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
import { LanderHostedEditor } from '@/domains/creative/lander_hosted_editor';
import { useBreadcrumbSegmentLabel } from '@/shell/breadcrumb_context';
import { useResource } from '@/api/use_resource';

export function LanderEditorPage() {
  const { id } = useParams();
  const landerId = id ?? '';
  const [reloadToken, setReloadToken] = useState(0);

  const { data, error, fetching } = useResource(
    (signal) => {
      if (!landerId) {
        return Promise.reject(new Error('Lander ID required'));
      }
      return getHostedEditorState(landerId, signal);
    },
    [landerId, reloadToken],
  );

  const [acting, setActing] = useState(false);
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
          setReloadToken((value) => value + 1);
        })
        .catch((err: unknown) => {
          setActionError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setActing(false);
        });
    },
    [landerId],
  );

  const onPublish = useCallback(() => {
    if (!landerId) {
      return;
    }
    setActing(true);
    setActionError(undefined);
    setActionMessage(undefined);
    void publishHostedLander(landerId)
      .then(() => {
        setActionMessage('Publish accepted.');
        toast.success('Publish accepted');
        setReloadToken((value) => value + 1);
      })
      .catch((err: unknown) => {
        setActionError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setActing(false);
      });
  }, [landerId]);

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
          setFileError(err instanceof Error ? err : new Error(String(err)));
        })
        .finally(() => {
          setFileLoading(false);
        });
    },
    [landerId],
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
        setReloadToken((value) => value + 1);
      })
      .catch((err: unknown) => {
        setFileError(err instanceof Error ? err : new Error(String(err)));
      })
      .finally(() => {
        setFileSaving(false);
      });
  }, [fileContent, landerId, selectedFilePath]);

  useBreadcrumbSegmentLabel(landerId || undefined, data?.name);

  return (
    <LanderHostedEditor
      landerId={landerId}
      state={data}
      fetching={fetching}
      error={error}
      hasSnapshot={data != null}
      acting={acting}
      actionError={actionError}
      actionMessage={actionMessage}
      selectedFilePath={selectedFilePath}
      fileContent={fileContent}
      fileLoading={fileLoading}
      fileSaving={fileSaving}
      fileError={fileError}
      onUploadZip={onUploadZip}
      onPublish={onPublish}
      onSelectFile={onSelectFile}
      onFileContentChange={setFileContent}
      onSaveFile={onSaveFile}
    />
  );
}
