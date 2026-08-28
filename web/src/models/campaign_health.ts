export type CampaignHealthLevel = 'ok' | 'warn' | 'risk' | 'paused';

export type CampaignHealthVM = {
  level: CampaignHealthLevel;
  label: string;
  title: string;
};

export function budgetUtilPct(campaign: {
  budget_limit?: string | number;
  current_spend?: string | number;
}): number | null {
  const limit = Number(campaign?.budget_limit ?? 0);
  const spend = Number(campaign?.current_spend ?? 0);
  if (!Number.isFinite(limit) || limit <= 0) return null;
  if (!Number.isFinite(spend) || spend < 0) return null;
  return (spend / limit) * 100;
}

export function attentionByCampaignId(
  attention: Array<{ id?: string; reason?: string }> | null | undefined
): Record<string, string> {
  const out: Record<string, string> = {};
  const rows = attention ?? [];
  for (let i = 0; i < rows.length; i++) {
    const row = rows[i];
    if (row?.id) out[row.id] = row.reason ?? '';
  }
  return out;
}

export type CampaignHealthContext = {
  portfolioRow?: {
    utilization_pct?: number | null;
    pacing_drift_pct?: number | null;
    overspend_risk?: boolean;
    margin_breach?: boolean;
  };
  attentionReason?: string;
  ivtElevated?: boolean;
  marginBreach?: boolean;
  licenseGrace?: boolean;
};

export function deriveCampaignHealth(
  campaign: { status?: string; budget_limit?: string | number; current_spend?: string | number },
  ctx: CampaignHealthContext = {}
): CampaignHealthVM {
  const status = String(campaign?.status ?? '').toUpperCase();
  if (status === 'PAUSED') {
    return {
      level: 'paused',
      label: 'Paused',
      title: ctx.attentionReason || 'Campaign is paused',
    };
  }

  const util = ctx.portfolioRow?.utilization_pct ?? budgetUtilPct(campaign);
  const drift = ctx.portfolioRow?.pacing_drift_pct;
  const reasons: string[] = [];
  let level: CampaignHealthLevel = 'ok';

  if (ctx.portfolioRow?.overspend_risk || (util != null && util >= 90)) {
    level = 'risk';
    reasons.push('Budget at or above 90%');
  } else if (util != null && util >= 75) {
    level = 'warn';
    reasons.push(`Budget ${util.toFixed(0)}% used`);
  }

  if (drift != null && Number(drift) >= 40) {
    if (level !== 'risk') level = 'warn';
    reasons.push(`Pacing drift ${Number(drift).toFixed(0)}%`);
  }

  if (ctx.ivtElevated) {
    level = 'risk';
    reasons.push('Elevated IVT (7d)');
  }

  if (ctx.marginBreach || ctx.portfolioRow?.margin_breach) {
    level = 'risk';
    reasons.push('Margin guard breach');
  }

  if (ctx.licenseGrace) {
    if (level === 'ok') level = 'warn';
    reasons.push('License in grace period');
  }

  if (ctx.attentionReason) {
    reasons.push(ctx.attentionReason);
    if (level === 'ok') level = 'warn';
  }

  if (reasons.length === 0) {
    return { level: 'ok', label: 'Healthy', title: 'Within budget and pacing targets' };
  }

  return {
    level,
    label: level === 'risk' ? 'At risk' : 'Watch',
    title: reasons.join(' , '),
  };
}
