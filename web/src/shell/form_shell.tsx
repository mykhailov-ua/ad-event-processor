import type { LucideIcon } from 'lucide-react';
import type { ComponentProps, ReactNode } from 'react';

import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { adminChrome } from '@/lib/admin_chrome';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export function FormSectionLabel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <p className={cn(adminTypography.label, className)}>{children}</p>;
}

export function InputWithIcon({
  icon: Icon,
  className,
  ...props
}: ComponentProps<'input'> & { icon: LucideIcon }) {
  return (
    <div className={cn(adminChrome.controlFieldGroup, 'min-w-0', className)}>
      <Icon aria-hidden className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={2} />
      <Input className={cn(adminChrome.controlFieldInset, 'w-full')} {...props} />
    </div>
  );
}

export function DashedActionZone({
  children,
  onClick,
  type = 'button',
  className,
}: {
  children: ReactNode;
  onClick?: () => void;
  type?: 'button' | 'submit';
  className?: string;
}) {
  return (
    <Button
      className={cn('h-auto min-h-7 w-full border-dashed py-4', className)}
      onClick={onClick}
      type={type}
      variant="outline"
    >
      {children}
    </Button>
  );
}
