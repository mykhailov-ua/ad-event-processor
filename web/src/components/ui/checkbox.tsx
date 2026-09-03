import * as React from 'react';
import { Check } from 'lucide-react';

import { cn } from '@/lib/utils';

export type CheckboxProps = Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type'> & {
  checked?: boolean;
  onCheckedChange?: (checked: boolean) => void;
};

const Checkbox = React.forwardRef<HTMLInputElement, CheckboxProps>(
  ({ className, checked, onCheckedChange, onChange, disabled, ...props }, ref) => (
    <span className="relative inline-flex h-4 w-4 shrink-0">
      <input
        ref={ref}
        checked={checked}
        className="peer sr-only"
        disabled={disabled}
        type="checkbox"
        onChange={(event) => {
          onChange?.(event);
          onCheckedChange?.(event.target.checked);
        }}
        {...props}
      />
      <span
        aria-hidden
        className={cn(
          'flex h-4 w-4 items-center justify-center rounded-sm border border-zinc-900 bg-white text-white transition-colors peer-focus-visible:outline-none peer-focus-visible:ring-2 peer-focus-visible:ring-zinc-400 peer-disabled:cursor-not-allowed peer-disabled:opacity-50 peer-checked:bg-zinc-900 dark:border-zinc-100 dark:bg-zinc-950 dark:peer-checked:bg-zinc-100 dark:peer-checked:text-zinc-900',
          className,
        )}
      >
        {checked ? <Check className="h-3 w-3" strokeWidth={3} /> : null}
      </span>
    </span>
  ),
);
Checkbox.displayName = 'Checkbox';

export { Checkbox };
