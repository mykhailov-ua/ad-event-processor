import type { FormHTMLAttributes, ReactNode } from 'react';

import { settingsFormStackClass } from '@/domains/settings/settings_classes';
import { cn } from '@/lib/utils';

export function SettingsFormStack({
  children,
  className,
  ...props
}: FormHTMLAttributes<HTMLFormElement> & { children: ReactNode }) {
  return (
    <form className={cn(settingsFormStackClass, className)} {...props}>
      {children}
    </form>
  );
}

export function SettingsFormActions({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return <div className={cn('flex flex-wrap items-center gap-2', className)}>{children}</div>;
}
