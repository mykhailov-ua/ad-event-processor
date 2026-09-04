import { formatAdminEnumLabel } from '@/lib/admin_typography';
import { cn } from '@/lib/utils';

export function opsStatusTone(status: string | undefined): string {
  const normalized = (status ?? '').toLowerCase();
  if (normalized === 'ok' || normalized === 'healthy' || normalized === 'pass') {
    return 'text-green-600 dark:text-green-400';
  }
  if (normalized === 'degraded' || normalized === 'warn' || normalized === 'warning') {
    return 'text-amber-600 dark:text-amber-400';
  }
  if (normalized === 'critical' || normalized === 'fail' || normalized === 'down') {
    return 'text-red-600 dark:text-red-400';
  }
  return 'text-muted-foreground';
}

export function OpsStatusChip({ status }: { status?: string }) {
  if (!status?.trim()) {
    return null;
  }
  return <span className={cn('text-xs text-muted-foreground', opsStatusTone(status))}>{formatAdminEnumLabel(status)}</span>;
}
