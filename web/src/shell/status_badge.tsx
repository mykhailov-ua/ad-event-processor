import { adminStatusBadgeBase, adminStatusBadgeClass, type AdminStatusTone } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type StatusBadgeProps = {
  label: string;
  tone?: AdminStatusTone;
  className?: string;
  title?: string;
};

export function StatusBadge({ label, tone = 'muted', className, title }: StatusBadgeProps) {
  return (
    <span className={cn(adminStatusBadgeBase, adminStatusBadgeClass[tone], className)} title={title ?? label}>
      {label}
    </span>
  );
}
