import * as React from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { cn } from '@/lib/utils';

const Input = React.forwardRef<HTMLInputElement, React.ComponentProps<'input'>>(
  ({ className, type, ...props }, ref) => (
    <input
      type={type}
      className={cn(adminChrome.control, 'w-full', className)}
      ref={ref}
      {...props}
    />
  ),
);
Input.displayName = 'Input';

export { Input };
