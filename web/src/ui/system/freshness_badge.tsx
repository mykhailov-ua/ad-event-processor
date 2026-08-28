import styles from './freshness_badge.module.css';

export type FreshnessBadgeProps = {
  stale?: boolean;
  chLagSeconds?: number;
  label?: string;
};

export function FreshnessBadge({ stale = false, chLagSeconds = 0, label }: FreshnessBadgeProps) {
  const tone = stale ? styles.stale : styles.fresh;
  const text =
    label ??
    (stale
      ? chLagSeconds > 0
        ? `Stale (${chLagSeconds}s lag)`
        : 'Stale'
      : chLagSeconds > 0
        ? `Fresh (${chLagSeconds}s lag)`
        : 'Fresh');

  return (
    <span className={`${styles.root} ${tone}`} data-stale={stale ? 'true' : 'false'}>
      {text}
    </span>
  );
}
