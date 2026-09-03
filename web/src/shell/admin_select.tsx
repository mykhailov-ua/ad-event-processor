import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { cn } from '@/lib/utils';

export type AdminSelectOption = {
  value: string;
  label: string;
};

export type AdminSelectProps = {
  'aria-label': string;
  className?: string;
  disabled?: boolean;
  title?: string;
  options: AdminSelectOption[];
  value: string;
  onValueChange?: (value: string) => void;
};

export function AdminSelect({
  'aria-label': ariaLabel,
  className,
  disabled = false,
  title,
  options,
  value,
  onValueChange,
}: AdminSelectProps) {
  return (
    <div className="inline-flex max-w-full">
      <Select disabled={disabled} value={value} onValueChange={onValueChange}>
      <SelectTrigger
        aria-label={ariaLabel}
        className={cn('flex h-8 w-full items-center justify-between rounded-md border border-zinc-200 bg-white px-3 text-sm dark:border-zinc-700 dark:bg-zinc-950 w-auto max-w-[12rem]', className)}
        title={title}
      >
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} plain value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
      </Select>
    </div>
  );
}
