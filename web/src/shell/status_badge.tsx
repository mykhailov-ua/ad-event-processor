import { adminStatusBadgeBase, adminStatusBadgeClass, type AdminStatusTone } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';

export type StatusBadgeProps = {
  label: string;
  tone?: AdminStatusTone;
  className?: string;
  title?: string;
  /** Match h-7 filter/action buttons in compact toolbars. */
  size?: 'default' | 'control';
};

export function StatusBadge({
  label,
  tone = 'muted',
  className,
  title,
  size = 'default',
}: StatusBadgeProps) {
  return (
    <span
      className={cn(
        adminStatusBadgeBase,
        adminStatusBadgeClass[tone],
        size === 'control' && 'h-7 min-h-7 items-center justify-center px-2.5 py-0',
        className
      )}
      title={title ?? label}
    >
      {label}
    </span>
  );
}
