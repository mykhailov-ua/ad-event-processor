import { lazy, Suspense, type ReactNode } from 'react';
import { Route, Routes } from 'react-router-dom';
import { GuardedRoute } from './components/guarded_route.js';
import { NotFoundPage } from './pages/not_found_page.js';
import { SIMPLE_REPORT_CONFIGS } from './pages/reports/report_configs.js';

const DevComponentsPage = lazy(() => import('./pages/dev_components_page.js').then((mod) => ({
  default: mod.DevComponentsPage,
})));
const CustomersPage = lazy(() => import('./pages/customers_page.js').then((mod) => ({
  default: mod.CustomersPage,
})));
const CampaignsPage = lazy(() => import('./pages/campaigns_page.js').then((mod) => ({
  default: mod.CampaignsPage,
})));
const RtbDealsPage = lazy(() => import('./pages/rtb_deals_page.js').then((mod) => ({
  default: mod.RtbDealsPage,
})));
const CustomerDetailPage = lazy(() => import('./pages/customer_detail_page.js').then((mod) => ({
  default: mod.CustomerDetailPage,
})));
const CampaignDetailPage = lazy(() => import('./pages/campaign_detail_page.js').then((mod) => ({
  default: mod.CampaignDetailPage,
})));
const InvoiceDetailPage = lazy(() => import('./pages/invoice_detail_page.js').then((mod) => ({
  default: mod.InvoiceDetailPage,
})));
const SettingsPage = lazy(() => import('./pages/settings_page.js').then((mod) => ({
  default: mod.SettingsPage,
})));
const SettingsLicensePage = lazy(() => import('./pages/settings_license_page.js').then((mod) => ({
  default: mod.SettingsLicensePage,
})));
const SettingsDomainsPage = lazy(() => import('./pages/settings_domains_page.js').then((mod) => ({
  default: mod.SettingsDomainsPage,
})));
const IntegrationsCostSyncPage = lazy(() => import('./pages/integrations_cost_sync_page.js').then((mod) => ({
  default: mod.IntegrationsCostSyncPage,
})));
const IntegrationsMarginGuardPage = lazy(() => import('./pages/integrations_margin_guard_page.js').then((mod) => ({
  default: mod.IntegrationsMarginGuardPage,
})));
const IntegrationsSmartAlertsPage = lazy(() => import('./pages/integrations_smart_alerts_page.js').then((mod) => ({
  default: mod.IntegrationsSmartAlertsPage,
})));
const IntegrationsSupplyPage = lazy(() => import('./pages/integrations_supply_page.js').then((mod) => ({
  default: mod.IntegrationsSupplyPage,
})));
const ReportsHubPage = lazy(() => import('./pages/reports/reports_hub_page.js').then((mod) => ({
  default: mod.ReportsHubPage,
})));
const ReportStubPage = lazy(() => import('./pages/reports/report_stub_page.js').then((mod) => ({
  default: mod.ReportStubPage,
})));
const PlacementsReportPage = lazy(() => import('./pages/reports/report_route_pages.js').then((mod) => ({
  default: mod.PlacementsReportPage,
})));
const KeywordsReportPage = lazy(() => import('./pages/reports/report_route_pages.js').then((mod) => ({
  default: mod.KeywordsReportPage,
})));
const IvtBySourceReportPage = lazy(() => import('./pages/reports/report_route_pages.js').then((mod) => ({
  default: mod.IvtBySourceReportPage,
})));
const TrafficSourcesReportPage = lazy(() => import('./pages/reports/report_route_pages.js').then((mod) => ({
  default: mod.TrafficSourcesReportPage,
})));
const GeoRoiReportPage = lazy(() => import('./pages/reports/report_route_pages.js').then((mod) => ({
  default: mod.GeoRoiReportPage,
})));
const TelegramSummaryPage = lazy(() => import('./pages/reports/telegram_summary_page.js').then((mod) => ({
  default: mod.TelegramSummaryPage,
})));
const TelegramFunnelPage = lazy(() => import('./pages/reports/telegram_funnel_page.js').then((mod) => ({
  default: mod.TelegramFunnelPage,
})));
const TelegramBotsPage = lazy(() => import('./pages/reports/telegram_bots_page.js').then((mod) => ({
  default: mod.TelegramBotsPage,
})));
const TelegramPremiumPage = lazy(() => import('./pages/reports/telegram_premium_page.js').then((mod) => ({
  default: mod.TelegramPremiumPage,
})));
const TelegramFraudPage = lazy(() => import('./pages/reports/telegram_fraud_page.js').then((mod) => ({
  default: mod.TelegramFraudPage,
})));

function lazySimpleReport(endpoint: string) {
  return lazy(() => import('./pages/reports/simple_report_page.js').then((mod) => {
    const config = SIMPLE_REPORT_CONFIGS.find((c) => c.endpoint === endpoint);
    if (!config) {
      throw new Error(`missing simple report config: ${endpoint}`);
    }
    const Page = () => (
      <mod.SimpleReportPage
        title={config.title}
        endpoint={config.endpoint}
        columns={config.columns}
      />
    );
    return { default: Page };
  }));
}

const SpendVelocityReportPage = lazySimpleReport('spend-velocity');
const DaypartHeatmapReportPage = lazySimpleReport('daypart-heatmap');
const CampaignGeoDeviceReportPage = lazySimpleReport('campaign-geo-device');
const SourceQualityReportPage = lazySimpleReport('source-quality');
const DiscrepancyBuySellReportPage = lazySimpleReport('discrepancy-buy-sell');
const TrueRoiReportPage = lazySimpleReport('true-roi');
const CampaignOverviewReportPage = lazySimpleReport('campaign-overview');
const CustomerPortfolioReportPage = lazySimpleReport('customer-portfolio');
const OpsHomePage = lazy(() => import('./pages/ops_home_page.js').then((mod) => ({
  default: mod.OpsHomePage,
})));
const OpsShardsPage = lazy(() => import('./pages/ops_shards_page.js').then((mod) => ({
  default: mod.OpsShardsPage,
})));
const OpsBlacklistPage = lazy(() => import('./pages/ops_blacklist_page.js').then((mod) => ({
  default: mod.OpsBlacklistPage,
})));
const OpsReconPage = lazy(() => import('./pages/ops_recon_page.js').then((mod) => ({
  default: mod.OpsReconPage,
})));
const BillingPage = lazy(() => import('./pages/billing_page.js').then((mod) => ({
  default: mod.BillingPage,
})));
const TeamPage = lazy(() => import('./pages/team_page.js').then((mod) => ({
  default: mod.TeamPage,
})));
const AuditPage = lazy(() => import('./pages/audit_page.js').then((mod) => ({
  default: mod.AuditPage,
})));
const OverviewPage = lazy(() => import('./pages/overview_page.js').then((mod) => ({
  default: mod.OverviewPage,
})));
const BuyerPortfolioPage = lazy(() => import('./pages/buyer_portfolio_page.js').then((mod) => ({
  default: mod.BuyerPortfolioPage,
})));
const CampaignTelegramPage = lazy(() => import('./pages/campaign_telegram_page.js').then((mod) => ({
  default: mod.CampaignTelegramPage,
})));
const RoleDashboardPage = lazy(() => import('./pages/role_dashboard_page.js').then((mod) => ({
  default: mod.RoleDashboardPage,
})));
const RtbIntegrationPage = lazy(() => import('./pages/rtb_integration_page.js').then((mod) => ({
  default: mod.RtbIntegrationPage,
})));

function RouteFallback() {
  return <span className="text-muted">Loading…</span>;
}

function lazyRoute(element: ReactNode) {
  return <Suspense fallback={<RouteFallback />}>{element}</Suspense>;
}

/**
 * Shell route table — all routes are React pages.
 */
export function AppRoutes() {
  return (
    <GuardedRoute>
      <Routes>
      <Route path="/" element={lazyRoute(<OverviewPage />)} />
      <Route path="/campaigns/portfolio" element={lazyRoute(<BuyerPortfolioPage />)} />
      <Route path="/campaigns/:id/telegram" element={lazyRoute(<CampaignTelegramPage />)} />
      <Route path="/dashboards/:role" element={lazyRoute(<RoleDashboardPage />)} />
      <Route path="/rtb/integration" element={lazyRoute(<RtbIntegrationPage />)} />
      <Route path="/dev/components" element={lazyRoute(<DevComponentsPage />)} />
      <Route path="/customers" element={lazyRoute(<CustomersPage />)} />
      <Route path="/customers/:id" element={lazyRoute(<CustomerDetailPage />)} />
      <Route path="/campaigns" element={lazyRoute(<CampaignsPage />)} />
      <Route path="/campaigns/:id" element={lazyRoute(<CampaignDetailPage />)} />
      <Route path="/rtb/deals" element={lazyRoute(<RtbDealsPage />)} />
      <Route path="/billing/invoices/:id" element={lazyRoute(<InvoiceDetailPage />)} />
      <Route path="/settings" element={lazyRoute(<SettingsPage />)} />
      <Route path="/settings/license" element={lazyRoute(<SettingsLicensePage />)} />
      <Route path="/settings/domains" element={lazyRoute(<SettingsDomainsPage />)} />
      <Route path="/integrations/cost-sync" element={lazyRoute(<IntegrationsCostSyncPage />)} />
      <Route path="/margin-guard" element={lazyRoute(<IntegrationsMarginGuardPage />)} />
      <Route path="/integrations/margin-guard" element={lazyRoute(<IntegrationsMarginGuardPage />)} />
      <Route path="/integrations/smart-alerts" element={lazyRoute(<IntegrationsSmartAlertsPage />)} />
      <Route path="/smart-alerts" element={lazyRoute(<IntegrationsSmartAlertsPage />)} />
      <Route path="/integrations/supply" element={lazyRoute(<IntegrationsSupplyPage />)} />
      <Route path="/reports" element={lazyRoute(<ReportsHubPage />)} />
      <Route path="/reports/placements" element={lazyRoute(<PlacementsReportPage />)} />
      <Route path="/reports/keywords" element={lazyRoute(<KeywordsReportPage />)} />
      <Route path="/reports/ivt-by-source" element={lazyRoute(<IvtBySourceReportPage />)} />
      <Route path="/reports/traffic-sources" element={lazyRoute(<TrafficSourcesReportPage />)} />
      <Route path="/reports/geo-roi" element={lazyRoute(<GeoRoiReportPage />)} />
      <Route path="/reports/spend-velocity" element={lazyRoute(<SpendVelocityReportPage />)} />
      <Route path="/reports/daypart-heatmap" element={lazyRoute(<DaypartHeatmapReportPage />)} />
      <Route path="/reports/campaign-geo-device" element={lazyRoute(<CampaignGeoDeviceReportPage />)} />
      <Route path="/reports/source-quality" element={lazyRoute(<SourceQualityReportPage />)} />
      <Route path="/reports/discrepancy-buy-sell" element={lazyRoute(<DiscrepancyBuySellReportPage />)} />
      <Route path="/reports/true-roi" element={lazyRoute(<TrueRoiReportPage />)} />
      <Route path="/reports/campaign-overview" element={lazyRoute(<CampaignOverviewReportPage />)} />
      <Route path="/reports/customer-portfolio" element={lazyRoute(<CustomerPortfolioReportPage />)} />
      <Route path="/reports/telegram" element={lazyRoute(<TelegramSummaryPage />)} />
      <Route path="/reports/telegram/funnel" element={lazyRoute(<TelegramFunnelPage />)} />
      <Route path="/reports/telegram/bots" element={lazyRoute(<TelegramBotsPage />)} />
      <Route path="/reports/telegram/premium" element={lazyRoute(<TelegramPremiumPage />)} />
      <Route path="/reports/telegram/fraud" element={lazyRoute(<TelegramFraudPage />)} />
      <Route path="/reports/:reportKey" element={lazyRoute(<ReportStubPage />)} />
      <Route path="/ops" element={lazyRoute(<OpsHomePage />)} />
      <Route path="/ops/shards" element={lazyRoute(<OpsShardsPage />)} />
      <Route path="/ops/blacklist" element={lazyRoute(<OpsBlacklistPage />)} />
      <Route path="/ops/recon" element={lazyRoute(<OpsReconPage />)} />
      <Route path="/billing" element={lazyRoute(<BillingPage />)} />
      <Route path="/team" element={lazyRoute(<TeamPage />)} />
      <Route path="/audit" element={lazyRoute(<AuditPage />)} />
      <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </GuardedRoute>
  );
}
