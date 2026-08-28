import { lazy, Suspense } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { GuardedRoute } from './ui/shell/guarded_route.js';
import { NotFoundPage } from './pages/not_found_page.js';
import { ReportRouteElements } from './report_routes.js';

const CustomersPage = lazy(() =>
  import('./pages/customers_page.js').then((mod) => ({
    default: mod.CustomersPage,
  }))
);
const CustomerDetailPage = lazy(() =>
  import('./pages/customer_detail_page.js').then((mod) => ({
    default: mod.CustomerDetailPage,
  }))
);
const CampaignDetailPage = lazy(() =>
  import('./pages/campaign_detail_page.js').then((mod) => ({
    default: mod.CampaignDetailPage,
  }))
);
const InvoiceDetailPage = lazy(() =>
  import('./pages/invoice_detail_page.js').then((mod) => ({
    default: mod.InvoiceDetailPage,
  }))
);
const FlowBuilderPage = lazy(() =>
  import('./pages/flow_builder_page.js').then((mod) => ({
    default: mod.FlowBuilderPage,
  }))
);
const RtbIntegrationPage = lazy(() =>
  import('./pages/rtb_integration_page.js').then((mod) => ({
    default: mod.RtbIntegrationPage,
  }))
);
const CampaignWizardPage = lazy(() =>
  import('./pages/campaign_wizard_page.js').then((mod) => ({
    default: mod.CampaignWizardPage,
  }))
);
const LanderEditorPage = lazy(() =>
  import('./pages/lander_editor_page.js').then((mod) => ({
    default: mod.LanderEditorPage,
  }))
);
const CampaignTelegramPage = lazy(() =>
  import('./pages/campaign_telegram_page.js').then((mod) => ({
    default: mod.CampaignTelegramPage,
  }))
);
const CampaignsPage = lazy(() =>
  import('./pages/campaigns_page.js').then((mod) => ({
    default: mod.CampaignsPage,
  }))
);
const CampaignFlowsPage = lazy(() =>
  import('./pages/campaign_flows_page.js').then((mod) => ({
    default: mod.CampaignFlowsPage,
  }))
);
const AuditPage = lazy(() =>
  import('./pages/audit_page.js').then((mod) => ({
    default: mod.AuditPage,
  }))
);
const BillingPage = lazy(() =>
  import('./pages/billing_page.js').then((mod) => ({
    default: mod.BillingPage,
  }))
);
const RtbDealsPage = lazy(() =>
  import('./pages/rtb_deals_page.js').then((mod) => ({
    default: mod.RtbDealsPage,
  }))
);
const BrandsPage = lazy(() =>
  import('./pages/brands_page.js').then((mod) => ({
    default: mod.BrandsPage,
  }))
);
const IntegrationsHubPage = lazy(() =>
  import('./pages/integrations_hub_page.js').then((mod) => ({
    default: mod.IntegrationsHubPage,
  }))
);
const IntegrationsCostSyncPage = lazy(() =>
  import('./pages/integrations_cost_sync_page.js').then((mod) => ({
    default: mod.IntegrationsCostSyncPage,
  }))
);
const IntegrationsPostbacksPage = lazy(() =>
  import('./pages/integrations_postbacks_page.js').then((mod) => ({
    default: mod.IntegrationsPostbacksPage,
  }))
);
const IntegrationsSchemasPage = lazy(() =>
  import('./pages/integrations_schemas_page.js').then((mod) => ({
    default: mod.IntegrationsSchemasPage,
  }))
);
const IntegrationsTemplatesImportPage = lazy(() =>
  import('./pages/integrations_templates_import_page.js').then((mod) => ({
    default: mod.IntegrationsTemplatesImportPage,
  }))
);
const IntegrationsSupplyPage = lazy(() =>
  import('./pages/integrations_supply_page.js').then((mod) => ({
    default: mod.IntegrationsSupplyPage,
  }))
);
const IntegrationsMarginGuardPage = lazy(() =>
  import('./pages/integrations_margin_guard_page.js').then((mod) => ({
    default: mod.IntegrationsMarginGuardPage,
  }))
);
const IntegrationsSmartAlertsPage = lazy(() =>
  import('./pages/integrations_smart_alerts_page.js').then((mod) => ({
    default: mod.IntegrationsSmartAlertsPage,
  }))
);
const IntegrationsAutomationPage = lazy(() =>
  import('./pages/integrations_automation_page.js').then((mod) => ({
    default: mod.IntegrationsAutomationPage,
  }))
);
const OpsHomePage = lazy(() =>
  import('./pages/ops_home_page.js').then((mod) => ({
    default: mod.OpsHomePage,
  }))
);
const OpsShardsPage = lazy(() =>
  import('./pages/ops_shards_page.js').then((mod) => ({
    default: mod.OpsShardsPage,
  }))
);
const OpsDlqPage = lazy(() =>
  import('./pages/ops_dlq_page.js').then((mod) => ({
    default: mod.OpsDlqPage,
  }))
);
const OpsDomainsPage = lazy(() =>
  import('./pages/ops_domains_page.js').then((mod) => ({
    default: mod.OpsDomainsPage,
  }))
);
const OpsBlacklistPage = lazy(() =>
  import('./pages/ops_blacklist_page.js').then((mod) => ({
    default: mod.OpsBlacklistPage,
  }))
);
const OpsReconPage = lazy(() =>
  import('./pages/ops_recon_page.js').then((mod) => ({
    default: mod.OpsReconPage,
  }))
);
const OpsConsentPage = lazy(() =>
  import('./pages/ops_consent_page.js').then((mod) => ({
    default: mod.OpsConsentPage,
  }))
);
const OpsMlModelPage = lazy(() =>
  import('./pages/ops_ml_model_page.js').then((mod) => ({
    default: mod.OpsMlModelPage,
  }))
);
const OpsEdgeParityPage = lazy(() =>
  import('./pages/ops_edge_parity_page.js').then((mod) => ({
    default: mod.OpsEdgeParityPage,
  }))
);
const SettingsPage = lazy(() =>
  import('./pages/settings_page.js').then((mod) => ({
    default: mod.SettingsPage,
  }))
);
const SettingsLicensePage = lazy(() =>
  import('./pages/settings_license_page.js').then((mod) => ({
    default: mod.SettingsLicensePage,
  }))
);
const SettingsDomainsPage = lazy(() =>
  import('./pages/settings_domains_page.js').then((mod) => ({
    default: mod.SettingsDomainsPage,
  }))
);
const SettingsDisputesPage = lazy(() =>
  import('./pages/settings_disputes_page.js').then((mod) => ({
    default: mod.SettingsDisputesPage,
  }))
);
const SettingsReportSchedulesPage = lazy(() =>
  import('./pages/settings_report_schedules_page.js').then((mod) => ({
    default: mod.SettingsReportSchedulesPage,
  }))
);
const TeamPage = lazy(() =>
  import('./pages/team_page.js').then((mod) => ({
    default: mod.TeamPage,
  }))
);
const SupportFeedbackPage = lazy(() =>
  import('./pages/support_feedback_page.js').then((mod) => ({
    default: mod.SupportFeedbackPage,
  }))
);
const SelfServeCampaignCreatePage = lazy(() =>
  import('./pages/selfserve_campaign_create_page.js').then((mod) => ({
    default: mod.SelfServeCampaignCreatePage,
  }))
);
const SelfServeHomePage = lazy(() =>
  import('./pages/selfserve_home_page.js').then((mod) => ({
    default: mod.SelfServeHomePage,
  }))
);
const SelfServeBillingPage = lazy(() =>
  import('./pages/selfserve_billing_page.js').then((mod) => ({
    default: mod.SelfServeBillingPage,
  }))
);
const SelfServeApiKeysPage = lazy(() =>
  import('./pages/selfserve_api_keys_page.js').then((mod) => ({
    default: mod.SelfServeApiKeysPage,
  }))
);
const PublisherPage = lazy(() =>
  import('./pages/publisher_page.js').then((mod) => ({
    default: mod.PublisherPage,
  }))
);
const FraudDecisionsPage = lazy(() =>
  import('./pages/fraud_decisions_page.js').then((mod) => ({
    default: mod.FraudDecisionsPage,
  }))
);
const FraudLabelsPage = lazy(() =>
  import('./pages/fraud_labels_page.js').then((mod) => ({
    default: mod.FraudLabelsPage,
  }))
);
const FraudOverridesPage = lazy(() =>
  import('./pages/fraud_overrides_page.js').then((mod) => ({
    default: mod.FraudOverridesPage,
  }))
);
const FraudPresetsPage = lazy(() =>
  import('./pages/fraud_presets_page.js').then((mod) => ({
    default: mod.FraudPresetsPage,
  }))
);
const FraudIntegrationsPage = lazy(() =>
  import('./pages/fraud_integrations_page.js').then((mod) => ({
    default: mod.FraudIntegrationsPage,
  }))
);

function RouteFallback() {
  return <span className="text-muted">Loading...</span>;
}

export function AppRoutes() {
  return (
    <GuardedRoute>
      <Routes>
        <Route path="/" element={<Navigate to="/customers" replace />} />
        <Route
          path="/customers"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CustomersPage />
            </Suspense>
          }
        />
        <Route
          path="/customers/:id"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CustomerDetailPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns/wizard"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CampaignWizardPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns/flows"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CampaignFlowsPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns/flows/:id/builder"
          element={
            <Suspense fallback={<RouteFallback />}>
              <FlowBuilderPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns/landers/:id/editor"
          element={
            <Suspense fallback={<RouteFallback />}>
              <LanderEditorPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns/:id/telegram"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CampaignTelegramPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns/:id"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CampaignDetailPage />
            </Suspense>
          }
        />
        <Route
          path="/campaigns"
          element={
            <Suspense fallback={<RouteFallback />}>
              <CampaignsPage />
            </Suspense>
          }
        />
        <Route
          path="/billing/invoices/:id"
          element={
            <Suspense fallback={<RouteFallback />}>
              <InvoiceDetailPage />
            </Suspense>
          }
        />
        <Route
          path="/rtb/integration"
          element={
            <Suspense fallback={<RouteFallback />}>
              <RtbIntegrationPage />
            </Suspense>
          }
        />
        <Route
          path="/audit"
          element={
            <Suspense fallback={<RouteFallback />}>
              <AuditPage />
            </Suspense>
          }
        />
        <Route path="/fraud" element={<Navigate to="/fraud/decisions" replace />} />
        <Route
          path="/fraud/decisions"
          element={
            <Suspense fallback={<RouteFallback />}>
              <FraudDecisionsPage />
            </Suspense>
          }
        />
        <Route
          path="/fraud/labels"
          element={
            <Suspense fallback={<RouteFallback />}>
              <FraudLabelsPage />
            </Suspense>
          }
        />
        <Route
          path="/fraud/overrides"
          element={
            <Suspense fallback={<RouteFallback />}>
              <FraudOverridesPage />
            </Suspense>
          }
        />
        <Route
          path="/fraud/presets"
          element={
            <Suspense fallback={<RouteFallback />}>
              <FraudPresetsPage />
            </Suspense>
          }
        />
        <Route
          path="/fraud/integrations"
          element={
            <Suspense fallback={<RouteFallback />}>
              <FraudIntegrationsPage />
            </Suspense>
          }
        />
        <Route
          path="/billing"
          element={
            <Suspense fallback={<RouteFallback />}>
              <BillingPage />
            </Suspense>
          }
        />
        <Route
          path="/rtb/deals"
          element={
            <Suspense fallback={<RouteFallback />}>
              <RtbDealsPage />
            </Suspense>
          }
        />
        <Route
          path="/brands"
          element={
            <Suspense fallback={<RouteFallback />}>
              <BrandsPage />
            </Suspense>
          }
        />
        <Route
          path="/integration/templates/import"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsTemplatesImportPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/cost-sync"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsCostSyncPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/postbacks"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsPostbacksPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/schemas"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsSchemasPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/supply"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsSupplyPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/margin-guard"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsMarginGuardPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/smart-alerts"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsSmartAlertsPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations/automation"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsAutomationPage />
            </Suspense>
          }
        />
        <Route
          path="/integrations"
          element={
            <Suspense fallback={<RouteFallback />}>
              <IntegrationsHubPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/shards"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsShardsPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/dlq"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsDlqPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/domains"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsDomainsPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/blacklist"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsBlacklistPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/recon"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsReconPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/consent"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsConsentPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/ml-model"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsMlModelPage />
            </Suspense>
          }
        />
        <Route
          path="/ops/edge-parity"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsEdgeParityPage />
            </Suspense>
          }
        />
        <Route
          path="/ops"
          element={
            <Suspense fallback={<RouteFallback />}>
              <OpsHomePage />
            </Suspense>
          }
        />
        <Route
          path="/settings/license"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SettingsLicensePage />
            </Suspense>
          }
        />
        <Route
          path="/settings/domains"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SettingsDomainsPage />
            </Suspense>
          }
        />
        <Route
          path="/settings/disputes"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SettingsDisputesPage />
            </Suspense>
          }
        />
        <Route
          path="/settings/report-schedules"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SettingsReportSchedulesPage />
            </Suspense>
          }
        />
        <Route
          path="/settings"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SettingsPage />
            </Suspense>
          }
        />
        <Route
          path="/team"
          element={
            <Suspense fallback={<RouteFallback />}>
              <TeamPage />
            </Suspense>
          }
        />
        <Route
          path="/selfserve/campaigns/new"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SelfServeCampaignCreatePage />
            </Suspense>
          }
        />
        <Route
          path="/selfserve/billing"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SelfServeBillingPage />
            </Suspense>
          }
        />
        <Route
          path="/selfserve/api-keys"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SelfServeApiKeysPage />
            </Suspense>
          }
        />
        <Route
          path="/selfserve"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SelfServeHomePage />
            </Suspense>
          }
        />
        <Route
          path="/publisher"
          element={
            <Suspense fallback={<RouteFallback />}>
              <PublisherPage />
            </Suspense>
          }
        />
        <Route
          path="/support/feedback"
          element={
            <Suspense fallback={<RouteFallback />}>
              <SupportFeedbackPage />
            </Suspense>
          }
        />
        <Route path="/margin-guard" element={<Navigate to="/integrations/margin-guard" replace />} />
        <Route
          path="/smart-alerts"
          element={<Navigate to="/integrations/smart-alerts" replace />}
        />
        {/*
          report-live-routes-gate paths:
          /reports /reports/placements /reports/keywords /reports/pacing-drift
          /reports/filter-rejects /reports/fraud-breakdown /reports/silent-reject-impression-funnel
          /reports/spend-velocity /reports/daypart-heatmap /reports/campaign-geo-device /reports/geo-roi
          /reports/source-quality /reports/ivt-by-source /reports/clicks /reports/conversion-type-payout
          /reports/postback-reconciliation /reports/rtb/overview /reports/rtb/no-bid-reasons
          /reports/rtb/geo-device /reports/traffic-sources /reports/discrepancy-buy-sell /reports/true-roi
          /reports/cost-sync-coverage /reports/campaign-overview /reports/customer-portfolio
          /reports/data-quality /reports/layer-desync-summary /reports/layer-desync-drilldown
          /reports/fraud-evidence-pack /reports/signal-effectiveness /reports/rtt-split-tunnel
          /reports/campaign-toggle-cohort /reports/wire-signal-breakdown /reports/customer-fraud-by-type
          /reports/customer-fraud-by-dimension /reports/customer-fraud-evidence /reports/edge-parity
          /reports/ml/feature-spikes /reports/ml/score-distribution /reports/ml/shadow-delta
          /reports/telegram /reports/telegram/funnel /reports/telegram/bots /reports/telegram/premium
          /reports/telegram/fraud /reports/ghost-impression-funnel
        */}
        {ReportRouteElements}
        <Route path="*" element={<NotFoundPage />} />
      </Routes>
    </GuardedRoute>
  );
}
