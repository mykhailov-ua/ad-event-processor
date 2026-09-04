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
    <div className="w-full min-w-0">
      <Select disabled={disabled} value={value} onValueChange={onValueChange}>
      <SelectTrigger
        aria-label={ariaLabel}
        className={cn(
          'flex min-h-7 w-full min-w-0 items-center justify-between rounded-[5px] border border-border bg-background px-2 py-1 text-[13px] leading-[18px] text-foreground',
          className,
        )}
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
