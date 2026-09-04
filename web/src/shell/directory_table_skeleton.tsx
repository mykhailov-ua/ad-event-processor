import { cn } from '@/lib/utils';

export type DirectoryTableSkeletonProps = {
  columns?: number;
  rows?: number;
  className?: string;
};

export function DirectoryTableSkeleton({
  columns = 4,
  rows = 6,
  className,
}: DirectoryTableSkeletonProps) {
  return (
    <div
      aria-busy="true"
      aria-label="Loading table"
      className={cn(
        'motion-safe:animate-pulse overflow-hidden rounded-md border border-border bg-card',
        className,
      )}
    >
      <div className="flex h-10 border-b border-border/40">
        {Array.from({ length: columns }, (_, index) => (
          <div key={`head-${index}`} className="flex flex-1 items-center px-2">
            <div className="h-3 w-20 rounded bg-muted" />
          </div>
        ))}
      </div>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <div key={`row-${rowIndex}`} className="flex border-b border-border/40 last:border-0">
          {Array.from({ length: columns }, (_, colIndex) => (
            <div key={`cell-${rowIndex}-${colIndex}`} className="flex flex-1 items-center p-2">
              <div
                className={cn(
                  'h-3 rounded bg-muted',
                  colIndex === 0 ? 'w-32' : colIndex === columns - 1 ? 'w-16' : 'w-24',
                )}
              />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
