import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import { Input } from '@/components/ui/input';
import { cn } from '@/lib/utils';

export function FormSectionLabel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <p className={cn('text-xs font-medium uppercase tracking-wide text-muted-foreground', className)}>
      {children}
    </p>
  );
}

export function InputWithIcon({
  icon: Icon,
  className,
  ...props
}: React.ComponentProps<typeof Input> & { icon: LucideIcon }) {
  return (
    <div className="relative">
      <Icon
        aria-hidden
        className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground"
        strokeWidth={2}
      />
      <Input className={cn('pl-9', className)} {...props} />
    </div>
  );
}

export function DashedActionZone({
  children,
  className,
  onClick,
  type = 'button',
}: {
  children: ReactNode;
  className?: string;
  onClick?: () => void;
  type?: 'button' | 'submit';
}) {
  return (
    <button
      className={cn(
        'flex w-full items-center justify-center rounded-2xl border border-dashed border-border/70 bg-muted/20 px-4 py-8 text-sm text-muted-foreground transition-colors hover:border-border hover:bg-muted/35 hover:text-foreground',
        className,
      )}
      onClick={onClick}
      type={type}
    >
      {children}
    </button>
  );
}
