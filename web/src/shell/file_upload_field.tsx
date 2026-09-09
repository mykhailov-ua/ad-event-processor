import { useRef, useState, type ChangeEvent } from 'react';

import { Button } from '@/components/ui/button';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { SecondaryActionButton } from '@/shell/action_buttons';

export type FileUploadFieldProps = {
  id: string;
  accept?: string;
  disabled?: boolean;
  multiple?: boolean;
  onFileChange: (file: File | undefined) => void;
  buttonLabel?: string;
  emptyLabel?: string;
  className?: string;
};

function formatSelectionLabel(files: FileList | null | undefined, multiple?: boolean): string | undefined {
  if (!files || files.length === 0) {
    return undefined;
  }
  if (multiple && files.length > 1) {
    return `${files.length} files selected`;
  }
  return files[0]?.name;
}

export function FileUploadField({
  id,
  accept,
  disabled,
  multiple,
  onFileChange,
  buttonLabel = 'Choose file',
  emptyLabel = 'No file selected',
  className,
}: FileUploadFieldProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [selectedLabel, setSelectedLabel] = useState<string | undefined>();

  function handleInputChange(event: ChangeEvent<HTMLInputElement>) {
    const files = event.target.files;
    const file = files?.[0];
    const label = formatSelectionLabel(files, multiple);
    setSelectedLabel(label);
    onFileChange(file);
  }

  function openPicker() {
    inputRef.current?.click();
  }

  function clearSelection() {
    if (inputRef.current) {
      inputRef.current.value = '';
    }
    setSelectedLabel(undefined);
    onFileChange(undefined);
  }

  const hasSelection = selectedLabel != null;

  return (
    <div className={cn('grid', adminSpacing.gap.md, className)}>
      <input
        ref={inputRef}
        id={id}
        type="file"
        accept={accept}
        disabled={disabled}
        multiple={multiple}
        className="sr-only"
        onChange={handleInputChange}
        tabIndex={-1}
      />
      <div className={cn('flex flex-wrap items-center', adminSpacing.gap.md)}>
        <Button disabled={disabled} type="button" variant="outline" onClick={openPicker}>
          {buttonLabel}
        </Button>
        {hasSelection ? (
          <SecondaryActionButton disabled={disabled} type="button" onClick={clearSelection}>
            Clear
          </SecondaryActionButton>
        ) : null}
      </div>
      <p className={cn('m-0', adminTypography.bodyMuted)}>
        {selectedLabel ?? emptyLabel}
      </p>
    </div>
  );
}
