import * as React from 'react';

import { cn } from '@/lib/utils';

export type SwitchProps = Omit<React.ButtonHTMLAttributes<HTMLButtonElement>, 'onChange'> & {
  checked: boolean;
  onCheckedChange?: (checked: boolean) => void;
};

const Switch = React.forwardRef<HTMLButtonElement, SwitchProps>(
  ({ checked, disabled, onCheckedChange, ...props }, ref) => (
    <button
      ref={ref}
      aria-checked={checked}
     
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
       
      />
    </button>
  )
);
Switch.displayName = 'Switch';

export { Switch };
