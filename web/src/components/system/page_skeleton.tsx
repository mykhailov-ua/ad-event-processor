import { DirectoryTableSkeleton } from '@/components/system/directory_table_skeleton';

export type PageSkeletonProps = {
  variant?: 'page' | 'directory';
  columns?: number;
  rows?: number;
};

export function PageSkeleton({
  variant = 'page',
  columns = 4,
  rows = 6,
}: PageSkeletonProps) {
  if (variant === 'directory') {
    return <DirectoryTableSkeleton columns={columns} rows={rows} />;
  }

  return (
    <div className="motion-safe:animate-pulse space-y-4 p-6" aria-busy="true" aria-label="Loading">
      <div className="h-8 w-48 rounded bg-muted" />
      <div className="h-64 rounded bg-muted" />
    </div>
  );
}
