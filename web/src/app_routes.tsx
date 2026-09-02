import { Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from '@/app_shell';
import { PageSkeleton } from '@/shell/page_skeleton';
import { useSession } from '@/hooks/use_session';
import { useMeta } from '@/hooks/use_meta';
import { defaultHomePath } from '@/lib/session';
import { CustomersPage } from '@/pages/customers_page';
import { CustomerDetailPage } from '@/pages/customer_detail_page';
import { DomainsPage } from '@/pages/domains_page';
import { FlowDetailPage } from '@/pages/flow_detail_page';
import { FlowsPage } from '@/pages/flows_page';
import { LanderEditorPage } from '@/pages/lander_editor_page';
import { LandersPage } from '@/pages/landers_page';
import { OffersPage } from '@/pages/offers_page';
import { SupplyAdsTxtPage } from '@/pages/supply_ads_txt_page';
import { SupplyPage } from '@/pages/supply_page';
import { SupplySellersPage } from '@/pages/supply_sellers_page';
import { CampaignEditorPage } from '@/pages/campaign_editor_page';
import { CampaignsPage } from '@/pages/campaigns_page';
import { BrandCreativesPage } from '@/pages/brand_creatives_page';
import { BrandsPage } from '@/pages/brands_page';
import { CreativeHubPage } from '@/pages/creative_hub_page';
import { BillingExportsPage } from '@/pages/billing_exports_page';
import { BillingPage } from '@/pages/billing_page';
import { InvoiceDetailPage } from '@/pages/invoice_detail_page';
import { ActivatePage } from '@/pages/activate_page';
import { ForbiddenPage } from '@/pages/forbidden_page';
import { InviteAcceptPage } from '@/pages/invite_accept_page';
import { LicenseSetupPage } from '@/pages/license_setup_page';
import { LoginPage } from '@/pages/login_page';
import { SetupPage } from '@/pages/setup_page';
import { NotFoundPage } from '@/pages/not_found_page';
import { AuditPage } from '@/pages/audit_page';
import { AutomationHubPage } from '@/pages/automation_hub_page';
import { AutomationPresetsPage } from '@/pages/automation_presets_page';
import { AutomationRulesPage } from '@/pages/automation_rules_page';
import { MarginGuardActivityPage } from '@/pages/margin_guard_activity_page';
import { MarginGuardPoliciesPage } from '@/pages/margin_guard_policies_page';
import { SmartAlertsHistoryPage } from '@/pages/smart_alerts_history_page';
import { SmartAlertsRulesPage } from '@/pages/smart_alerts_rules_page';
import { TrafficOptimizerPresetsPage } from '@/pages/traffic_optimizer_presets_page';
import { TrafficOptimizerRulesPage } from '@/pages/traffic_optimizer_rules_page';
import { DashboardPage } from '@/pages/dashboard_page';
import { FraudDecisionPage } from '@/pages/fraud_decision_page';
import { IntegrationsAffiliatePresetsPage } from '@/pages/integrations_affiliate_presets_page';
import { IntegrationsCostSyncPage } from '@/pages/integrations_cost_sync_page';
import { IntegrationsHubPage } from '@/pages/integrations_hub_page';
import { IntegrationsPlatformCampaignsPage } from '@/pages/integrations_platform_campaigns_page';
import { IntegrationsPostbacksPage } from '@/pages/integrations_postbacks_page';
import { IntegrationsSchemasPage } from '@/pages/integrations_schemas_page';
import { FraudHubPage } from '@/pages/fraud_hub_page';
import { FraudIntegrationsPage } from '@/pages/fraud_integrations_page';
import { FraudLabelsPage } from '@/pages/fraud_labels_page';
import { FraudOverridesPage } from '@/pages/fraud_overrides_page';
import { FraudPresetsPage } from '@/pages/fraud_presets_page';
import { OpsBlacklistPage } from '@/pages/ops_blacklist_page';
import { OpsConsentPage } from '@/pages/ops_consent_page';
import { OpsDlqPage } from '@/pages/ops_dlq_page';
import { OpsDomainsPage } from '@/pages/ops_domains_page';
import { OpsIncidentsPage } from '@/pages/ops_incidents_page';
import { OpsMetricsPage } from '@/pages/ops_metrics_page';
import { OpsMlModelPage } from '@/pages/ops_ml_model_page';
import { OpsOutboxPage } from '@/pages/ops_outbox_page';
import { OpsPage } from '@/pages/ops_page';
import { OpsReconPage } from '@/pages/ops_recon_page';
import { OpsRumPage } from '@/pages/ops_rum_page';
import { OpsShardsPage } from '@/pages/ops_shards_page';
import { ReportJobsPage } from '@/pages/report_jobs_page';
import { ClickLogPage } from '@/pages/click_log_page';
import { ReportRunnerPage } from '@/pages/report_runner_page';
import { ReportsPage } from '@/pages/reports_page';
import { RtbPage } from '@/pages/rtb_page';
import { RtbDealEditorPage } from '@/pages/rtb_deal_editor_page';
import { RtbDealsPage } from '@/pages/rtb_deals_page';
import { RtbFloorsPage } from '@/pages/rtb_floors_page';
import { RtbIntegrationProfilePage } from '@/pages/rtb_integration_profile_page';
import { RtbShadowPage } from '@/pages/rtb_shadow_page';
import { RtbValidatePage } from '@/pages/rtb_validate_page';
import { CampaignForecastPage } from '@/pages/campaign_forecast_page';
import { PortalsHubPage } from '@/pages/portals_hub_page';
import { PublisherDashboardPage } from '@/pages/publisher_dashboard_page';
import { PublisherStatementsPage } from '@/pages/publisher_statements_page';
import { ReportSchedulesPage } from '@/pages/report_schedules_page';
import { SavedViewsPage } from '@/pages/saved_views_page';
import { SelfServePortalPage } from '@/pages/selfserve_portal_page';
import { TelegramBotsPage } from '@/pages/telegram_bots_page';
import { TelegramBotEditorPage } from '@/pages/telegram_bot_editor_page';
import { TelegramPostbacksPage } from '@/pages/telegram_postbacks_page';
import { CampaignDashboardPage } from '@/pages/campaign_dashboard_page';
import { DisputesPage } from '@/pages/disputes_page';
import { SupportFeedbackPage } from '@/pages/support_feedback_page';
import { SettingsLicensePage } from '@/pages/settings_license_page';
import { SettingsPage } from '@/pages/settings_page';
import { TeamPage } from '@/pages/team_page';
import { DocsPage } from '@/pages/docs_page';
import { RouteErrorPage } from '@/pages/route_error_page';

function ProtectedLayout() {
  const { bootstrapComplete, licenseNeedsSetup, loading: metaLoading } = useMeta();
  const { authenticated, forbidden, loading: sessionLoading } = useSession();

  if (metaLoading || sessionLoading) {
    return <PageSkeleton />;
  }

  if (!bootstrapComplete) {
    return <Navigate replace to="/setup" />;
  }

  if (forbidden) {
    return <Navigate replace to="/forbidden" />;
  }

  if (!authenticated) {
    return <Navigate replace to="/login" />;
  }

  if (licenseNeedsSetup) {
    return <LicenseSetupPage />;
  }

  return <AppShell />;
}

function HomeRedirect() {
  const { session, loading } = useSession();

  if (loading || !session) {
    return <PageSkeleton />;
  }

  return <Navigate replace to={defaultHomePath(session)} />;
}

export function AppRoutes() {
  return (
    <Routes>
      <Route errorElement={<RouteErrorPage layout="standalone" />}>
        <Route element={<SetupPage />} path="/setup" />
        <Route element={<LoginPage />} path="/login" />
        <Route element={<ActivatePage />} path="/activate" />
        <Route element={<InviteAcceptPage />} path="/invite/accept" />
        <Route element={<ForbiddenPage />} path="/forbidden" />
        <Route element={<Navigate replace to="/settings/license" />} path="/licence" />
        <Route element={<Navigate replace to="/settings/license" />} path="/license" />
        <Route element={<Navigate replace to="/settings/license" />} path="/settings/licence" />
        <Route element={<ProtectedLayout />} errorElement={<RouteErrorPage layout="embedded" />}>
        <Route element={<HomeRedirect />} index />
        <Route element={<CustomersPage />} path="customers" />
        <Route element={<CustomerDetailPage />} path="customers/:id" />
        <Route element={<CampaignsPage />} path="campaigns" />
        <Route element={<CampaignEditorPage />} path="campaigns/:id/edit" />
        <Route element={<BillingPage />} path="billing" />
        <Route element={<InvoiceDetailPage />} path="billing/invoices/:id" />
        <Route element={<BillingExportsPage />} path="billing/exports" />
        <Route element={<OpsPage />} path="ops" />
        <Route element={<OpsDlqPage />} path="ops/dlq" />
        <Route element={<OpsBlacklistPage />} path="ops/blacklist" />
        <Route element={<OpsIncidentsPage />} path="ops/incidents" />
        <Route element={<OpsOutboxPage />} path="ops/outbox" />
        <Route element={<OpsShardsPage />} path="ops/shards" />
        <Route element={<OpsMlModelPage />} path="ops/ml-model" />
        <Route element={<OpsDomainsPage />} path="ops/domains" />
        <Route element={<OpsReconPage />} path="ops/recon" />
        <Route element={<OpsConsentPage />} path="ops/consent" />
        <Route element={<OpsRumPage />} path="ops/rum" />
        <Route element={<OpsMetricsPage />} path="ops/metrics" />
        <Route element={<SettingsPage />} path="settings" />
        <Route element={<SettingsLicensePage />} path="settings/license" />
        <Route element={<SupportFeedbackPage />} path="support/feedback" />
        <Route element={<DisputesPage />} path="disputes" />
        <Route element={<TeamPage />} path="team" />
        <Route element={<AuditPage />} path="audit" />
        <Route element={<CampaignDashboardPage />} path="dashboards/campaign/:id" />
        <Route element={<DashboardPage />} path="dashboards/:role" />
        <Route element={<ReportsPage />} path="reports" />
        <Route element={<ReportJobsPage />} path="reports/jobs" />
        <Route element={<ClickLogPage />} path="reports/click-log" />
        <Route element={<ReportRunnerPage />} path="reports/:key" />
        <Route element={<RtbPage />} path="rtb" />
        <Route element={<RtbDealsPage />} path="rtb/deals" />
        <Route element={<RtbDealEditorPage />} path="rtb/deals/:id" />
        <Route element={<RtbShadowPage />} path="rtb/shadow" />
        <Route element={<RtbFloorsPage />} path="rtb/floors" />
        <Route element={<RtbValidatePage />} path="rtb/validate" />
        <Route element={<RtbIntegrationProfilePage />} path="rtb/integration-profile" />
        <Route element={<FraudHubPage />} path="fraud" />
        <Route element={<FraudIntegrationsPage />} path="fraud/integrations" />
        <Route element={<FraudLabelsPage />} path="fraud/labels" />
        <Route element={<FraudOverridesPage />} path="fraud/overrides" />
        <Route element={<FraudPresetsPage />} path="fraud/presets" />
        <Route element={<FraudDecisionPage />} path="fraud/decisions" />
        <Route element={<IntegrationsHubPage />} path="integrations" />
        <Route element={<IntegrationsCostSyncPage />} path="integrations/cost-sync" />
        <Route element={<IntegrationsPostbacksPage />} path="integrations/postbacks" />
        <Route element={<IntegrationsSchemasPage />} path="integrations/schemas" />
        <Route element={<IntegrationsPlatformCampaignsPage />} path="integrations/platform-campaigns" />
        <Route element={<IntegrationsAffiliatePresetsPage />} path="integrations/affiliate-presets" />
        <Route element={<CreativeHubPage />} path="creative" />
        <Route element={<FlowsPage />} path="flows" />
        <Route element={<FlowDetailPage />} path="flows/:id" />
        <Route element={<LandersPage />} path="landers" />
        <Route element={<LanderEditorPage />} path="landers/:id/editor" />
        <Route element={<OffersPage />} path="offers" />
        <Route element={<BrandsPage />} path="brands" />
        <Route element={<BrandCreativesPage />} path="brand-creatives/:id" />
        <Route element={<SupplyPage />} path="supply" />
        <Route element={<SupplySellersPage />} path="supply/sellers" />
        <Route element={<SupplyAdsTxtPage />} path="supply/ads-txt" />
        <Route element={<DomainsPage />} path="domains" />
        <Route element={<AutomationHubPage />} path="automation" />
        <Route element={<AutomationPresetsPage />} path="automation/presets" />
        <Route element={<AutomationRulesPage />} path="automation/rules" />
        <Route element={<TrafficOptimizerPresetsPage />} path="traffic-optimizer/presets" />
        <Route element={<TrafficOptimizerRulesPage />} path="traffic-optimizer/rules" />
        <Route element={<SmartAlertsRulesPage />} path="smart-alerts/rules" />
        <Route element={<SmartAlertsHistoryPage />} path="smart-alerts/history" />
        <Route element={<MarginGuardPoliciesPage />} path="margin-guard/policies" />
        <Route element={<MarginGuardActivityPage />} path="margin-guard/activity" />
        <Route element={<PortalsHubPage />} path="portals" />
        <Route element={<SelfServePortalPage />} path="selfserve" />
        <Route element={<PublisherDashboardPage />} path="publisher/dashboard" />
        <Route element={<PublisherStatementsPage />} path="publisher/statements" />
        <Route element={<TelegramBotsPage />} path="telegram/bots" />
        <Route element={<TelegramBotEditorPage />} path="telegram/bots/:campaignId" />
        <Route element={<TelegramPostbacksPage />} path="telegram/postbacks" />
        <Route element={<Navigate replace to="/telegram/bots" />} path="telegram" />
        <Route element={<ReportSchedulesPage />} path="report-schedules" />
        <Route element={<SavedViewsPage />} path="views" />
        <Route element={<CampaignForecastPage />} path="forecast/campaign" />
        <Route element={<DocsPage />} path="docs/*" />
        <Route element={<NotFoundPage />} path="*" />
        </Route>
        <Route element={<NotFoundPage />} path="*" />
      </Route>
    </Routes>
  );
}
