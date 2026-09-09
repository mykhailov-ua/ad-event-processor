import * as React from 'react';
import { Eye, EyeOff } from 'lucide-react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

const PasswordInput = React.forwardRef<HTMLInputElement, React.ComponentProps<'input'>>(
  ({ className, disabled, 'aria-invalid': ariaInvalid, ...props }, ref) => {
    const [visible, setVisible] = React.useState(false);

    return (
      <div
        aria-invalid={ariaInvalid}
        className={cn(adminChrome.controlFieldGroup, 'w-full pr-1', className)}
      >
        <input
          ref={ref}
          type={visible ? 'text' : 'password'}
          className={adminChrome.controlFieldInset}
          disabled={disabled}
          aria-invalid={ariaInvalid}
          {...props}
        />
        <button
          type="button"
          className={cn(
            'inline-flex h-6 w-6 shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground focus-visible:outline-none focus-visible:ring-0 disabled:pointer-events-none disabled:opacity-50',
            adminKit.nestedRadius
          )}
          aria-label={visible ? 'Hide password' : 'Show password'}
          aria-pressed={visible}
          disabled={disabled}
          onClick={() => setVisible((current) => !current)}
        >
          {visible ? (
            <EyeOff aria-hidden className="h-4 w-4" />
          ) : (
            <Eye aria-hidden className="h-4 w-4" />
          )}
        </button>
      </div>
    );
  }
);
PasswordInput.displayName = 'PasswordInput';

export { PasswordInput };
