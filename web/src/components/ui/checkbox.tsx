import * as React from 'react';
import { Check } from 'lucide-react';

import { adminKit } from '@/lib/admin_kit';
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
        className="peer absolute inset-0 z-10 h-full w-full cursor-pointer opacity-0"
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
          'pointer-events-none flex h-4 w-4 items-center justify-center border border-muted-foreground/70 bg-card text-primary-foreground transition-colors peer-focus-visible:outline-none peer-focus-visible:ring-2 peer-focus-visible:ring-primary/30 peer-disabled:cursor-not-allowed peer-disabled:border-input peer-disabled:bg-admin-input-disabled peer-disabled:opacity-100 peer-checked:border-primary peer-checked:bg-primary',
          adminKit.nestedRadius,
          className
        )}
      >
        {checked ? <Check className="h-3 w-3" strokeWidth={3} /> : null}
      </span>
    </span>
  )
);
Checkbox.displayName = 'Checkbox';

export { Checkbox };
