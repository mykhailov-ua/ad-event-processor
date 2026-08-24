export type FreshnessBadgeProps = {
  stale?: boolean;
  lagSeconds?: number;
  sources?: Array<{
    name: string;
    consistency: string;
    stale?: boolean;
    ch_lag_seconds?: number;
  }>;
};

function sourceLabel(name: string, consistency: string): string {
  if (name === 'counts') {
    return consistency === 'strong' ? 'Counts (PG strong)' : `Counts (${consistency})`;
  }
  if (name === 'money') {
    return consistency === 'eventual' ? 'Money (CH eventual)' : `Money (${consistency})`;
  }
  return `${name} (${consistency})`;
}

function SourceBadge({
  name,
  consistency,
  stale = false,
  lagSeconds = 0,
}: {
  name: string;
  consistency: string;
  stale?: boolean;
  lagSeconds?: number;
}) {
  const lagText = lagSeconds > 0 ? `${lagSeconds}s lag` : 'no lag data';
  const title = stale
    ? `${sourceLabel(name, consistency)} may be stale. ${lagText}.`
    : `${sourceLabel(name, consistency)} is fresh. ${lagText}.`;
  return (
    <span
      className={
        stale ? 'freshness-badge freshness-badge--stale' : 'freshness-badge freshness-badge--ok'
      }
      title={title}
      data-testid={`freshness-badge-${name}`}
    >
      {sourceLabel(name, consistency)}
      {stale ? ` , stale` : ''}
    </span>
  );
}

export function FreshnessBadge({ stale = false, lagSeconds = 0, sources }: FreshnessBadgeProps) {
  if (sources?.length) {
    return (
      <div className="cluster cluster--xs" data-testid="freshness-badges">
        {sources.map((source) => (
          <SourceBadge
            key={source.name}
            name={source.name}
            consistency={source.consistency}
            stale={source.stale}
            lagSeconds={source.ch_lag_seconds ?? 0}
          />
        ))}
      </div>
    );
  }

  const lag = lagSeconds ?? 0;
  if (!stale && lag === 0) return null;

  const lagText = lag > 0 ? `${lag}s lag` : 'no lag data';
  const title = stale
    ? `ClickHouse data may be stale. ${lagText}.`
    : `ClickHouse data is fresh. ${lagText}.`;

  return (
    <span
      className={
        stale ? 'freshness-badge freshness-badge--stale' : 'freshness-badge freshness-badge--ok'
      }
      title={title}
    >
      {stale ? `Stale , ${lagText}` : `Fresh , ${lagText}`}
    </span>
  );
}
