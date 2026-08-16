export type FreshnessBadgeProps = {
  stale?: boolean;
  lagSeconds?: number;
};

/**
 * ClickHouse data freshness indicator when lag or staleness is known.
 */
export function FreshnessBadge({ stale = false, lagSeconds = 0 }: FreshnessBadgeProps) {
  const lag = lagSeconds ?? 0;
  if (!stale && lag === 0) return null;

  const lagText = lag > 0 ? `${lag}s lag` : 'no lag data';
  const title = stale
    ? `ClickHouse data may be stale. ${lagText}.`
    : `ClickHouse data is fresh. ${lagText}.`;

  return (
    <span
      className={stale ? 'freshness-badge freshness-badge--stale' : 'freshness-badge freshness-badge--ok'}
      title={title}
    >
      {stale ? `Stale · ${lagText}` : `Fresh · ${lagText}`}
    </span>
  );
}
