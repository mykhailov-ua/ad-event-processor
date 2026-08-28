import { cn } from '../../lib/cn.js';
import styles from './page_skeleton.module.css';

export type PageSkeletonProps = {
  rows?: number;
  columns?: number;
  className?: string;
};

export function PageSkeleton({ rows = 3, columns = 4, className }: PageSkeletonProps) {
  const colTemplate = `repeat(${Math.max(1, columns)}, minmax(0, 1fr))`;

  return (
    <div
      className={cn(styles.root, className)}
      aria-busy="true"
      aria-label="Loading page content"
    >
      <div className={styles.header}>
        <span className={cn(styles.bar, styles.barTitle)} />
        <span className={cn(styles.bar, styles.barSubtitle)} />
      </div>
      <div className={styles.grid}>
        {Array.from({ length: rows }, (_, rowIndex) => (
          <div
            key={`skel-row-${rowIndex}`}
            className={styles.row}
            style={{ gridTemplateColumns: colTemplate }}
          >
            {Array.from({ length: columns }, (__, colIndex) => (
              <span
                key={`skel-cell-${rowIndex}-${colIndex}`}
                className={cn(styles.bar, styles.cell, colIndex === columns - 1 ? styles.cellShort : '')}
              />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}
