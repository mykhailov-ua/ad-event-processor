import type { BuyerPortfolio } from '@/domains/dashboards/buyer_dashboard_types';

const MICRO_PER_USD = 1_000_000;

export function formatDashboardUsdFromMicro(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  const usd = value / MICRO_PER_USD;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(usd);
}

export function formatDashboardRoiPct(value: number): string {
  if (!Number.isFinite(value)) {
    return '';
  }
  const sign = value > 0 ? '+' : '';
  return `${sign}${value.toFixed(2)}%`;
}

export function formatDashboardCrPct(value?: number | null): string {
  if (value == null || !Number.isFinite(value)) {
    return '';
  }
  return `${value.toFixed(2)}%`;
}

export function portfolioCostMicro(portfolio: BuyerPortfolio): number | undefined {
  const kpis = portfolio.kpis;
  if (kpis?.cost_micro != null) {
    return kpis.cost_micro;
  }
  return kpis?.spend_micro;
}

export function resolvePortfolioProfitMicro(portfolio: BuyerPortfolio): number | undefined {
  const kpis = portfolio.kpis;
  const costMicro = portfolioCostMicro(portfolio);
  const revenueMicro = kpis?.revenue_micro;
  if (costMicro != null && revenueMicro != null) {
    return revenueMicro - costMicro;
  }
  if (kpis?.profit_micro != null) {
    return kpis.profit_micro;
  }
  return undefined;
}

export function derivePortfolioRoiPct(portfolio: BuyerPortfolio): number | undefined {
  const costMicro = portfolioCostMicro(portfolio) ?? 0;
  if (costMicro <= 0) {
    return portfolio.kpis?.roi_pct;
  }
  const profitMicro = resolvePortfolioProfitMicro(portfolio) ?? 0;
  return (profitMicro / costMicro) * 100;
}
