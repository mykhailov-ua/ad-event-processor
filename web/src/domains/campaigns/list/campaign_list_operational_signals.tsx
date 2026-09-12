import type { CampaignListMetrics } from '@/api/campaigns_api';
import type { Campaign } from '@/api/types';
import { formatBudgetUsedPercent } from '@/lib/campaign_budget_used';
import { adminTypography } from '@/lib/admin_kit';
import { cn } from '@/lib/utils';
import { Badge } from '@/components/ui/badge';

export type CampaignPacingHealth = 'ok' | 'drift' | 'exhausted';

function pacingHealthLabel(health: CampaignPacingHealth | undefined): string | undefined {
  switch (health) {
    case 'drift':
      return 'Pacing drift';
    case 'exhausted':
      return 'Budget exhausted';
    case 'ok':
      return undefined;
    default:
      return undefined;
  }
}

function pacingHealthTone(health: CampaignPacingHealth | undefined): string {
  switch (health) {
    case 'drift':
      return 'border-amber-500/40 bg-amber-500/10 text-amber-900 dark:text-amber-100';
    case 'exhausted':
      return 'border-destructive/40 bg-destructive/10 text-destructive';
    default:
      return '';
  }
}

export function CampaignListOperationalSignals({
  campaign,
  metrics,
  className,
}: {
  campaign: Campaign;
  metrics?: CampaignListMetrics;
  className?: string;
}) {
  const health = metrics?.pacing_health as CampaignPacingHealth | undefined;
  const healthLabel = pacingHealthLabel(health);
  const burnPct =
    metrics?.budget_burn_pct ??
    (typeof campaign.budget_used_pct === 'number' ? campaign.budget_used_pct : undefined);
  const burnLabel =
    burnPct != null && Number.isFinite(burnPct) ? formatBudgetUsedPercent(burnPct) : undefined;
  const stale = metrics?.metrics_stale === true || metrics?.stale === true;
  const pacingMode = metrics?.pacing_mode ?? campaign.pacing_mode;

  const hasSignals = healthLabel || burnLabel || stale || pacingMode;
  if (!hasSignals) {
    return null;
  }

  return (
    <span className={cn('inline-flex flex-wrap items-center gap-1.5', className)}>
      {healthLabel ? (
        <Badge className={cn('font-normal', pacingHealthTone(health))} variant="outline">
          {healthLabel}
        </Badge>
      ) : null}
      {burnLabel ? (
        <span className={cn(adminTypography.captionPlain, 'text-muted-foreground')}>
          {burnLabel} burn
        </span>
      ) : null}
      {pacingMode ? (
        <span className={cn(adminTypography.captionPlain, 'text-muted-foreground')}>
          {pacingMode}
        </span>
      ) : null}
      {stale ? (
        <span className={cn(adminTypography.captionPlain, 'text-amber-700 dark:text-amber-300')}>
          Stale metrics
        </span>
      ) : null}
    </span>
  );
}

export function CampaignListNameWithSignals({
  campaign,
  metrics,
}: {
  campaign: Campaign;
  metrics?: CampaignListMetrics;
}) {
  const name = campaign.name ?? campaign.id ?? '';
  return (
    <span className="grid gap-1">
      <span>{name}</span>
      <CampaignListOperationalSignals campaign={campaign} metrics={metrics} />
    </span>
  );
}
