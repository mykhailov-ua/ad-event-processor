import {
  dashboardKpiGridClass,
  dashboardKpiTileClass,
} from '@/domains/dashboards/dashboard_classes';
import { cn } from '@/lib/utils';

export type DashboardKpiTile = {
  id: string;
  label: string;
  value: string;
  accent?: 1 | 2 | 3 | 4 | 5;
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
        <div key={tile.id} className={dashboardKpiTileClass}>
          <p className="whitespace-nowrap text-ui-caption tracking-wide text-muted-foreground sm:text-xs">
            {tile.label}
          </p>
          <p className="whitespace-nowrap font-numeric text-base font-semibold tabular-nums tracking-tight sm:text-lg">
            {tile.value}
          </p>
        </div>
      ))}
    </div>
  );
}
