import * as React from 'react';
import { Loader2 } from 'lucide-react';

import { buttonVariantClass, type ButtonVariant } from '@/lib/admin_chrome';
import { Slot } from '@/lib/as_child';
import { cn } from '@/lib/utils';

export type ButtonProps = React.ButtonHTMLAttributes<HTMLButtonElement> & {
  asChild?: boolean;
  loading?: boolean;
  variant?: ButtonVariant;
  size?: 'default' | 'sm' | 'lg' | 'icon';
  shape?: 'default' | 'pill' | 'square';
};

const sizeClass: Record<NonNullable<ButtonProps['size']>, string> = {
  default: '',
  sm: 'h-7 px-2 text-xs',
  lg: 'h-10 px-5 text-sm',
  icon: 'h-8 w-8 p-0',
};

const shapeClass: Record<NonNullable<ButtonProps['shape']>, string> = {
  default: '',
  pill: 'rounded-full',
  square: 'rounded-sm',
};

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant = 'default',
      size = 'default',
      shape = 'default',
      asChild = false,
      loading = false,
      disabled,
      children,
      type = 'button',
      ...props
    },
    ref,
  ) => {
    const classes = cn(
      'inline-flex min-h-7 items-center justify-center gap-2 rounded-[5px] border px-2.5 py-1 text-[13px] font-semibold leading-[18px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50',
      buttonVariantClass[variant],
      sizeClass[size],
      shapeClass[shape],
      className,
    );

    if (asChild) {
      return (
        <Slot
          ref={ref}
          aria-busy={loading || undefined}
          aria-disabled={disabled || loading || undefined}
          className={classes}
          {...props}
        >
          {children}
        </Slot>
      );
    }

    return (
      <button
        className={classes}
        ref={ref}
        disabled={disabled || loading}
        aria-busy={loading || undefined}
        type={type}
        {...props}
      >
        {loading ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : null}
        {children}
      </button>
    );
  },
);
Button.displayName = 'Button';

export function buttonVariants({
  variant = 'default',
  size = 'default',
  shape = 'default',
}: {
  variant?: ButtonVariant;
  size?: NonNullable<ButtonProps['size']>;
  shape?: NonNullable<ButtonProps['shape']>;
} = {}) {
  return cn(
    'inline-flex min-h-7 items-center justify-center gap-2 rounded-[5px] border px-2.5 py-1 text-[13px] font-semibold leading-[18px] transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50',
    buttonVariantClass[variant],
    sizeClass[size],
    shapeClass[shape],
  );
}

export { Button };
