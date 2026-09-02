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
        className={cn('admin-select-trigger w-auto max-w-[12rem]', className)}
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
