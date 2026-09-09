import { X } from 'lucide-react';
import type { ReactNode } from 'react';

import { adminAlertClass, adminKit, type AdminAlertTone } from '@/lib/admin_kit';
import { uiSurfaces } from '@/lib/ui_surfaces';
import { cn } from '@/lib/utils';

export type AdminAlertProps = {
  tone: AdminAlertTone;
  title: string;
  description?: ReactNode;
  onDismiss?: () => void;
};

export function AdminAlert({ tone, title, description, onDismiss }: AdminAlertProps) {
  return (
    <div
     
      role="alert"
    >
      <div >
        <p >{title}</p>
        {description ? <p >{description}</p> : null}
      </div>
      {onDismiss ? (
        <button
          aria-label="Dismiss"
         
          type="button"
          onClick={onDismiss}
        >
          <X aria-hidden  />
        </button>
      ) : null}
    </div>
  );
}
