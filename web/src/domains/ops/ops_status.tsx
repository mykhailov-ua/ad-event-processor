import { formatAdminEnumLabel } from '@/lib/admin_typography';
import { cn } from '@/lib/utils';

export function opsStatusTone(status: string | undefined): string {
  const normalized = (status ?? '').toLowerCase();
  if (normalized === 'ok' || normalized === 'healthy' || normalized === 'pass') {
    return 'admin-ops-status--ok';
  }
  if (normalized === 'degraded' || normalized === 'warn' || normalized === 'warning') {
    return 'admin-ops-status--warn';
  }
  if (normalized === 'critical' || normalized === 'fail' || normalized === 'down') {
    return 'admin-ops-status--fail';
  }
  return 'admin-ops-status--muted';
}

export function OpsStatusChip({ status }: { status?: string }) {
  if (!status?.trim()) {
    return null;
  }
  return <span className={cn('admin-stat-note', opsStatusTone(status))}>{formatAdminEnumLabel(status)}</span>;
}
