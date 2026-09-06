import type { LucideIcon } from 'lucide-react';
import type { ReactNode } from 'react';

import { adminChrome } from '@/lib/admin_chrome';
import { adminKit } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export function FormSectionLabel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <p className={cn('text-xs font-medium tracking-wide text-muted-foreground', className)}>
      {children}
    </p>
  );
}

export function InputWithIcon({
  icon: Icon,
  className,
  ...props
}: React.ComponentProps<'input'> & { icon: LucideIcon }) {
  return (
    <div className={cn(adminChrome.controlFieldGroup, 'min-w-0')}>
      <Icon aria-hidden className="size-4 shrink-0 text-muted-foreground" strokeWidth={2} />
      <input
        className={cn(adminChrome.controlFieldInset, adminKit.controlText, className)}
        {...props}
      />
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
        'flex w-full items-center justify-center border border-dashed border-border/70 bg-muted/20 px-4 py-8 text-sm text-muted-foreground transition-colors hover:border-border hover:bg-muted/35 hover:text-foreground',
        adminKit.panelRadius,
        className
      )}
      onClick={onClick}
      type={type}
    >
      {children}
    </button>
  );
}
