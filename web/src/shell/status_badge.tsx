import { adminStatusBadgeBase, adminStatusBadgeClass, type AdminStatusTone } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type StatusBadgeProps = {
  label: string;
  tone?: AdminStatusTone;
  title?: string;
};

export function StatusBadge({ label, tone = 'muted', title }: StatusBadgeProps) {
  return (
    <span
     
      title={title ?? label}
    >
      {label}
    </span>
  );
}
