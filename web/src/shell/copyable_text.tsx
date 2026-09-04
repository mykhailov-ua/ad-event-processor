import { CopyButton } from '@/shell/copy_button';
import { cn } from '@/lib/utils';

export type CopyableTextProps = {
  className?: string;
  label?: string;
  mono?: boolean;
  title?: string;
  value: string;
};

export function CopyableText({
  className,
  label,
  mono = false,
  title,
  value,
}: CopyableTextProps) {
  const trimmed = value.trim();
  if (!trimmed) {
    return <span className={className}>-</span>;
  }

  return (
    <span className={cn('inline-flex min-w-0 max-w-full items-center gap-0.5', className)}>
      <span
        className={cn('min-w-0 select-text truncate', mono && 'font-mono text-xs tabular-nums')}
        title={title ?? trimmed}
      >
        {trimmed}
      </span>
      <CopyButton className="size-6 shrink-0" label={label ?? 'Value'} value={trimmed} />
    </span>
  );
}
