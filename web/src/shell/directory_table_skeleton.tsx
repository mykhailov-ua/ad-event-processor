import { cn } from '@/lib/utils';

export type DirectoryTableSkeletonProps = {
  columns?: number;
  rows?: number;
};

export function DirectoryTableSkeleton({
  columns = 4,
  rows = 6,
}: DirectoryTableSkeletonProps) {
  return (
    <div
      aria-busy="true"
      aria-label="Loading table"
     
    >
      <div >
        {Array.from({ length: columns }, (_, index) => (
          <div key={`head-${index}`}>
            <div  />
          </div>
        ))}
      </div>
      {Array.from({ length: rows }, (_, rowIndex) => (
        <div
          key={`row-${rowIndex}`}
         
        >
          {Array.from({ length: columns }, (_, colIndex) => (
            <div key={`cell-${rowIndex}-${colIndex}`}>
              <div
               
              />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
