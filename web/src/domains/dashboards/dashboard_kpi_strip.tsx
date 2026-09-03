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
      className={cn(
        'grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7',
        className,
      )}
      role="region"
    >
      {tiles.map((tile) => (
        <div
          key={tile.id}
          className="ui-surface-raised grid min-w-0 gap-0.5 px-3 py-2.5 text-center sm:px-4 sm:py-3"
        >
          <p className="truncate text-[11px] tracking-wide text-muted-foreground sm:text-xs">
            {tile.label}
          </p>
          <p className="truncate font-numeric text-base font-semibold tabular-nums tracking-tight sm:text-lg">
            {tile.value}
          </p>
        </div>
      ))}
    </div>
  );
}
