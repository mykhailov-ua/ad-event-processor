import {
  adminKpiAccentSurfaceClass,
  adminKpiAccentTopBarClass,
  adminKpiAccentValueClass,
  type AdminKpiAccent,
} from '@/lib/admin_metric_tone';
import {
  dashboardKpiGridClass,
  dashboardKpiTileClass,
} from '@/domains/dashboards/dashboard_classes';
import { cn } from '@/lib/utils';

export type DashboardKpiTile = {
  id: string;
  label: string;
  value: string;
  accent?: AdminKpiAccent;
};

export type DashboardKpiStripProps = {
  tiles: DashboardKpiTile[];
  className?: string;
};

export function DashboardKpiStrip({ tiles, className }: DashboardKpiStripProps) {
  if (tiles.length === 0) {
    return null;
  }

  return (
    <div
      aria-label="Key performance indicators"
      className={cn(dashboardKpiGridClass, className)}
      role="region"
    >
      {tiles.map((tile) => (
        <div
          key={tile.id}
          className={cn(
            dashboardKpiTileClass,
            tile.accent ? adminKpiAccentSurfaceClass[tile.accent] : null,
            tile.accent ? adminKpiAccentTopBarClass[tile.accent] : null
          )}
        >
          <p className="whitespace-nowrap text-ui-caption tracking-wide text-muted-foreground sm:text-xs">
            {tile.label}
          </p>
          <p
            className={cn(
              'whitespace-nowrap  text-base font-semibold tracking-tight sm:text-lg',
              tile.accent ? adminKpiAccentValueClass[tile.accent] : 'text-foreground'
            )}
          >
            {tile.value}
          </p>
        </div>
      ))}
    </div>
  );
}
