import { CopyButton } from '@/shell/copy_button';
import { cn } from '@/lib/utils';

export type CopyableTextProps = {
  label?: string;
  mono?: boolean;
  title?: string;
  value: string;
};

export function CopyableText({ label, mono = false, title, value }: CopyableTextProps) {
  const trimmed = value.trim();
  if (!trimmed) {
    return <span >-</span>;
  }

  return (
    <span >
      <span
       
        title={title ?? trimmed}
      >
        {trimmed}
      </span>
      <CopyButton label={label ?? 'Value'} value={trimmed} />
    </span>
  );
}
