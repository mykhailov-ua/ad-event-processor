import * as React from 'react';

import { cn } from '@/lib/utils';

export type SwitchProps = Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> & {
  checked: boolean;
  onCheckedChange?: (checked: boolean) => void;
};

const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(
  ({ checked, className, disabled, onCheckedChange, ...props }, ref) => (
    <button
      ref={ref}
      aria-checked={checked}
      className={cn(
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background',
        checked
          ? 'border-primary bg-primary'
          : 'border-border/80 bg-muted/30',
        disabled ? 'cursor-not-allowed' : 'cursor-pointer',
        className,
      )}
      disabled={disabled}
      onClick={() => {
        if (!disabled) {
          onCheckedChange?.(!checked);
        }
      }}
      role="switch"
      type="button"
      {...props}
    >
      <span
        className={cn(
          'pointer-events-none block h-3.5 w-3.5 rounded-full shadow-sm transition-transform',
          checked ? 'translate-x-4 bg-background' : 'translate-x-0.5 bg-muted-foreground/75',
        )}
      />
    </button>
  ),
);
Switch.displayName = 'Switch';

export { Switch };
