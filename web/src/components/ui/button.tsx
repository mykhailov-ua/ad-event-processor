import * as React from 'react';
import { Loader2 } from 'lucide-react';

import { buttonVariantClass, type ButtonVariant } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
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
  sm: 'text-xs',
  lg: 'px-5',
  icon: 'size-7 p-0',
};

const shapeClass: Record<NonNullable<ButtonProps['shape']>, string> = {
  default: '',
  pill: adminKit.pillRadius,
  square: adminKit.controlRadius,
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
    ref
  ) => {
    const classes = cn(
      adminKit.buttonShell,
      adminKit.controlRadius,
      'font-normal transition-colors focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0 disabled:pointer-events-none disabled:opacity-50',
      buttonVariantClass[variant],
      sizeClass[size],
      shapeClass[shape],
      className
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
  }
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
    adminKit.buttonShell,
    adminKit.controlRadius,
    'font-normal transition-colors focus-visible:outline-none focus-visible:ring-0 focus-visible:ring-offset-0 disabled:pointer-events-none disabled:opacity-50',
    buttonVariantClass[variant],
    sizeClass[size],
    shapeClass[shape]
  );
}

export { Button };
