import styles from './loading_count_badge.module.css';

export type LoadingCountBadgeProps = {
  loading: boolean;
  label: string;
};

export function LoadingCountBadge({ loading, label }: LoadingCountBadgeProps) {
  return (
    <span className={styles.root} aria-busy={loading || undefined}>
      {loading ? '\u2026' : label}
    </span>
  );
}
