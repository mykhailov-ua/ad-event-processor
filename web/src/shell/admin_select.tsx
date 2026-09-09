import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/ui/select';

export type AdminSelectOption = {
  value: string;
  label: string;
};

export type AdminSelectProps = {
  'aria-label': string;
  disabled?: boolean;
  title?: string;
  options: AdminSelectOption[];
  value: string;
  onValueChange?: (value: string) => void;
};

export function AdminSelect({
  'aria-label': ariaLabel,
  disabled = false,
  title,
  options,
  value,
  onValueChange,
}: AdminSelectProps) {
  const selectedLabel = options.find((option) => option.value === value)?.label;

  return (
    <Select disabled={disabled} value={value} onValueChange={onValueChange}>
      <SelectTrigger aria-label={ariaLabel} title={title}>
        <span className="truncate text-left">{selectedLabel ?? 'Select...'}</span>
      </SelectTrigger>
      <SelectContent>
        {options.map((option) => (
          <SelectItem key={option.value} plain value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
