import type { ReactNode } from 'react';

import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { adminSpacing, adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type DirectoryOverviewField = {
  label: string;
  value: ReactNode;
};

export type DirectoryOverviewDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  title: ReactNode;
  fields: DirectoryOverviewField[];
  footer?: ReactNode;
  children?: ReactNode;
};

export function DirectoryOverviewDialog({
  open,
  onOpenChange,
  title,
  fields,
  footer,
  children,
}: DirectoryOverviewDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>
        <dl className={cn('grid', adminSpacing.gap.md)}>
          {fields.map((field) => (
            <div key={field.label} className={cn('grid', adminSpacing.gap.xs)}>
              <dt className={adminTypography.labelMuted}>{field.label}</dt>
              <dd className={cn('m-0', adminTypography.body)}>{field.value}</dd>
            </div>
          ))}
        </dl>
        {children}
        {footer ? <DialogFooter>{footer}</DialogFooter> : null}
      </DialogContent>
    </Dialog>
  );
}
