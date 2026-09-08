import type { BuyerPortfolio } from '@/domains/dashboards/buyer_dashboard_types';
import {
  resolveEconomicsProfitMicro,
  resolveEconomicsRoiPct,
  type EconomicsMicroRow,
} from '@/lib/economics';

export { formatDashboardCrPct, formatDashboardRoiPct } from '@/lib/display_metrics';

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

export function resolveBreakdownProfitMicro(row: EconomicsMicroRow): number | undefined {
  return resolveEconomicsProfitMicro(row);
}

export function resolveBreakdownRoiPct(row: EconomicsMicroRow): number | undefined {
  return resolveEconomicsRoiPct(row);
}

export function derivePortfolioRoiPct(portfolio: BuyerPortfolio): number | undefined {
  const costMicro = portfolioCostMicro(portfolio) ?? 0;
  if (costMicro <= 0) {
    return portfolio.kpis?.roi_pct;
  }
  const profitMicro = resolvePortfolioProfitMicro(portfolio) ?? 0;
  return (profitMicro / costMicro) * 100;
}
