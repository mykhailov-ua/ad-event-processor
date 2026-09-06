import * as React from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

const numberInputClass =
  '[appearance:textfield] [-moz-appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none';

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<'input'>>(
  ({ className, type, ...props }, ref) => (
    <input
      type={type}
      className={cn(
        adminChrome.control,
        'w-full',
        type === 'number' && numberInputClass,
        className
      )}
      ref={ref}
      {...props}
    />
  )
);
Input.displayName = 'Input';

export { Input };
