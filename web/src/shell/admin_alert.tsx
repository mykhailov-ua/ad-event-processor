import { X } from 'lucide-react';
import type { ReactNode } from 'react';

import { adminAlertClass, adminKit, type AdminAlertTone } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

export type AdminAlertProps = {
  tone: AdminAlertTone;
  title: string;
  description?: ReactNode;
  className?: string;
  onDismiss?: () => void;
};

export function AdminAlert({ tone, title, description, className, onDismiss }: AdminAlertProps) {
  return (
    <div
      className={cn(
        uiSurfaces.message,
        adminAlertClass[tone],
        'grid-cols-[1fr_auto] items-start',
        adminKit.controlRadius,
        className
      )}
      role="alert"
    >
      <div className="flex min-w-0 flex-col gap-1">
        <p className="m-0 font-semibold">{title}</p>
        {description ? <p className="m-0 opacity-90">{description}</p> : null}
      </div>
      {onDismiss ? (
        <button
          aria-label="Dismiss"
          className="shrink-0 rounded p-0.5 opacity-70 transition-opacity hover:opacity-100"
          type="button"
          onClick={onDismiss}
        >
          <X aria-hidden className="h-4 w-4" />
        </button>
      ) : null}
    </div>
  );
}
