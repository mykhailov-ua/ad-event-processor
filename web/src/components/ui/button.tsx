import * as React from 'react';
import { Slot } from '@radix-ui/react-slot';
import { cva, type VariantProps } from 'class-variance-authority';
import { Loader2 } from 'lucide-react';

import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'admin-btn inline-flex items-center justify-center gap-2 font-medium transition-all duration-150 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50 disabled:transform-none',
  {
    variants: {
      variant: {
        default: 'admin-btn--primary',
        destructive: 'border-destructive/30 bg-destructive text-destructive-foreground hover:bg-destructive/90',
        outline: 'border-border bg-background hover:bg-secondary hover:text-secondary-foreground',
        secondary: 'border-transparent bg-secondary text-secondary-foreground hover:bg-secondary/80',
        ghost: 'border-transparent bg-transparent hover:bg-secondary hover:text-secondary-foreground',
        link: 'border-0 bg-transparent text-primary underline-offset-4 hover:underline',
      },
      size: {
        default: '',
        sm: '',
        lg: 'px-5 text-sm',
        icon: 'admin-btn--icon',
      },
      shape: {
        default: '',
        pill: 'rounded-full',
        square: 'rounded-[var(--admin-radius-sm)]',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
      shape: 'default',
    },
  },
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
  loading?: boolean;
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      className,
      variant,
      size,
      shape,
      asChild = false,
      loading = false,
      disabled,
      children,
      ...props
    },
    ref,
  ) => {
    const Comp = asChild ? Slot : 'button';
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, shape, className }))}
        ref={ref}
        disabled={disabled || loading}
        aria-busy={loading || undefined}
        {...props}
      >
        {loading ? <Loader2 className="h-4 w-4" aria-hidden="true" /> : null}
        {children}
      </Comp>
    );
  },
);
Button.displayName = 'Button';

export { Button, buttonVariants };
