import type { ReactNode } from 'react';

import {
  settingsCardBodyClass,
  settingsCardClass,
  settingsCardHeaderClass,
  settingsCardTitleClass,
} from '@/domains/settings/settings_classes';
import { cn } from '@/lib/utils';

export function SettingsCard({
  bodyClassName,
  children,
  className,
  meta,
  title,
}: {
  bodyClassName?: string;
  children: ReactNode;
  className?: string;
  meta?: ReactNode;
  title: ReactNode;
}) {
  return (
    <section className={cn(settingsCardClass, className)}>
      <div className={settingsCardHeaderClass}>
        <h2 className={settingsCardTitleClass}>{title}</h2>
        {meta}
      </div>
      <div className={cn(settingsCardBodyClass, bodyClassName)}>{children}</div>
    </section>
  );
}
