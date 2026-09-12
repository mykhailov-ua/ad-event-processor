import { LanderHostedEditor } from '@/domains/creative/lander_hosted_editor';
import { LanderWysiwygPanel } from '@/domains/creative/lander_wysiwyg_panel';
import { useLanderHostedEditorPageWorkspace } from '@/domains/creative/use_lander_hosted_editor_page_workspace';

export function LanderHostedEditorPage() {
  const workspace = useLanderHostedEditorPageWorkspace();
  const {
    landerId,
    canWrite,
    state,
    fetching,
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
    onSelectFile,
    fileContent,
    onFileContentChange,
    fileLoading,
    fileSaving,
    fileError,
    onUploadZip,
    onPublish,
    onSaveFile,
  } = workspace;

  return (
    <LanderHostedEditor
      acting={acting}
      actionError={actionError}
      actionMessage={actionMessage}
      error={error}
      fetching={fetching}
      fileContent={fileContent}
      fileError={fileError}
      fileLoading={fileLoading}
      fileSaving={fileSaving}
      hasSnapshot={hasSnapshot}
      landerId={landerId}
      selectedFilePath={selectedFilePath}
      state={state}
      wysiwyg={
        <LanderWysiwygPanel
          blocks={blocks}
          dirty={blocksDirty}
          disabled={!canWrite}
          error={blocksError}
          previewHtml={blockPreviewHtml}
          saving={blocksSaving}
          onAddBlock={onAddBlock}
          onBlocksChange={onBlocksChange}
          onSave={onSaveBlocks}
        />
      }
      onFileContentChange={onFileContentChange}
      onPublish={onPublish}
      onSaveFile={onSaveFile}
      onSelectFile={onSelectFile}
      onUploadZip={onUploadZip}
    />
  );
}
