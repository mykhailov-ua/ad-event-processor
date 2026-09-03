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
        'relative inline-flex h-5 w-9 shrink-0 items-center rounded-full border transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-zinc-400 focus-visible:ring-offset-2 focus-visible:ring-offset-white dark:focus-visible:ring-offset-zinc-950',
        checked
          ? 'border-zinc-900 bg-zinc-900 dark:border-zinc-100 dark:bg-zinc-100'
          : 'border-zinc-300 bg-zinc-100 dark:border-zinc-600 dark:bg-zinc-800',
        disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer',
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
          'pointer-events-none block h-3.5 w-3.5 rounded-full transition-transform',
          checked
            ? 'translate-x-4 bg-white dark:bg-zinc-900'
            : 'translate-x-0.5 bg-zinc-500 dark:bg-zinc-300',
        )}
      />
    </button>
  ),
);
Switch.displayName = 'Switch';

export { Switch };
