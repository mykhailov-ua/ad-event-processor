import { reportHubPath } from '@/lib/report_paths';

export type FraudHubReportLink = {
  path: string;
  title: string;
  description: string;
  meta: string;
  typed?: boolean;
};

export const FRAUD_HUB_REPORT_LINKS: FraudHubReportLink[] = [
  {
    path: reportHubPath('fraud-breakdown'),
    title: 'Fraud reasons',
    description: 'Fraud events by reason and placement with CSV export.',
    meta: 'Typed report',
    typed: true,
  },
  {
    path: reportHubPath('wire-signal-breakdown'),
    title: 'Wire fraud signals',
    description: 'L7, TLS, and HTTP/2 wire fraud signals.',
    meta: 'Typed report',
    typed: true,
  },
  {
    path: reportHubPath('customer-fraud-by-type'),
    title: 'Fraud by type',
    description: 'Customer-facing fraud categories, shares, and silent-reject ratios.',
    meta: 'Typed report',
    typed: true,
  },
  {
    path: reportHubPath('silent-reject-impression-funnel'),
    title: 'Non-blocking response funnel',
    description: 'Billable vs non-blocking fraud response vs IVT impressions.',
    meta: 'Catalog report',
  },
  {
    path: reportHubPath('signal-effectiveness'),
    title: 'Signal effectiveness',
    description: 'Wire signal block and silent-reject rates.',
    meta: 'Catalog report',
  },
  {
    path: reportHubPath('customer-fraud-by-dimension'),
    title: 'Fraud by dimension',
    description: 'Fraud concentration by placement, geo, or sub.',
    meta: 'Catalog report',
  },
  {
    path: reportHubPath('ivt-by-source'),
    title: 'IVT by source',
    description: 'Invalid traffic by sub and geo.',
    meta: 'Catalog report',
  },
  {
    path: reportHubPath('postback-reconciliation'),
    title: 'Postback reconciliation',
    description: 'Postback vs ledger reconciliation.',
    meta: 'Billing report',
  },
  {
    path: reportHubPath('layer-desync-summary'),
    title: 'Layer desync summary',
    description: 'Cross-layer desync fraud counts by campaign.',
    meta: 'Operator report',
  },
  {
    path: reportHubPath('filter-rejects'),
    title: 'Filter rejects',
    description: 'Unified filter reject kinds.',
    meta: 'Ingress report',
  },
];
