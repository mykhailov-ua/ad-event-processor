import { adminTypography } from '@/lib/admin_kit';
import {
  adminOpsCriticalClass,
  adminOpsHealthyClass,
  adminOpsWarnClass,
} from '@/lib/admin_metric_tone';
import { formatAdminEnumLabel } from '@/lib/admin_typography';
import { cn } from '@/lib/utils';

export function opsStatusTone(status: string | undefined): string {
  const normalized = (status ?? '').toLowerCase();
  if (normalized === 'ok' || normalized === 'healthy' || normalized === 'pass') {
    return adminOpsHealthyClass;
  }
  if (normalized === 'degraded' || normalized === 'warn' || normalized === 'warning') {
    return adminOpsWarnClass;
  }
  if (normalized === 'critical' || normalized === 'fail' || normalized === 'down') {
    return adminOpsCriticalClass;
  }
  return 'text-muted-foreground';
}

export function OpsStatusChip({ status }: { status?: string }) {
  if (!status?.trim()) {
    return null;
  }
  return (
    <span className={cn(adminTypography.captionPlain, opsStatusTone(status))}>
      {formatAdminEnumLabel(status)}
    </span>
  );
}
