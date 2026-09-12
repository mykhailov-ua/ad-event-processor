import { DirectoryTableSkeleton } from '@/shell/directory_table_skeleton';

export type PageSkeletonProps = {
  variant?: 'page' | 'directory';
  columns?: number;
  rows?: number;
};

export function PageSkeleton({ variant = 'page', columns = 4, rows = 6 }: PageSkeletonProps) {
  if (variant === 'directory') {
    return <DirectoryTableSkeleton columns={columns} rows={rows} />;
  }

  return (
    <div aria-busy="true" aria-label="Loading">
      <div />
      <div />
    </div>
  );
}
