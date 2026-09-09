import { Navigate, Route, Routes } from 'react-router-dom';

import { AppShell } from '@/app_shell';
import { PageSkeleton } from '@/shell/page_skeleton';
import { useSession } from '@/hooks/use_session';
import { useMeta } from '@/hooks/use_meta';
import { defaultHomePath } from '@/lib/session';
import { CustomersPage } from '@/pages/customers_page';
import { CustomerDetailPage } from '@/pages/customer_detail_page';
import { CampaignEditorPage } from '@/pages/campaign_editor_page';
import { CampaignsPage } from '@/pages/campaigns_page';
import { InvoiceDetailPage } from '@/pages/invoice_detail_page';
import { ActivatePage } from '@/pages/activate_page';
import { ForbiddenPage } from '@/pages/forbidden_page';
import { InviteAcceptPage } from '@/pages/invite_accept_page';
import { LicenseSetupPage } from '@/pages/license_setup_page';
import { LoginPage } from '@/pages/login_page';
import { SetupPage } from '@/pages/setup_page';
import { NotFoundPage } from '@/pages/not_found_page';
import { AuditPage } from '@/pages/audit_page';
import { ExportsPage } from '@/pages/exports_page';
import { CampaignIdRedirect } from '@/shell/campaign_id_redirect';
import {
  BillingExportsRoute,
  PreserveSearchRedirect,
  ReportJobsRoute,
} from '@/shell/export_hub_legacy_redirect';
import { ReportIndexLegacyRedirect, ReportLegacyRedirect } from '@/shell/report_legacy_redirect';
import { IntegrationsAffiliatePresetsPage } from '@/pages/integrations_affiliate_presets_page';
import { IntegrationsCostSyncPage } from '@/pages/integrations_cost_sync_page';
import { IntegrationsHubPage } from '@/pages/integrations_hub_page';
import { IntegrationsPlatformCampaignsPage } from '@/pages/integrations_platform_campaigns_page';
import { IntegrationsApiKeysPage } from '@/pages/integrations_api_keys_page';
import { IntegrationsPostbacksPage } from '@/pages/integrations_postbacks_page';
import { IntegrationsDebuggerPage } from '@/pages/integrations_debugger_page';
import { IntegrationsSchemasPage } from '@/pages/integrations_schemas_page';
import { OpsBlacklistPage } from '@/pages/ops_blacklist_page';
import { OpsConsentPage } from '@/pages/ops_consent_page';
import { OpsSyncErrorsPage } from '@/pages/ops_sync_errors_page';
import { OpsDomainsPage } from '@/pages/ops_domains_page';
import { OpsIncidentsPage } from '@/pages/ops_incidents_page';
import { OpsMetricsPage } from '@/pages/ops_metrics_page';
import { OpsMlModelPage } from '@/pages/ops_ml_model_page';
import { OpsOutboxPage } from '@/pages/ops_outbox_page';
import { OpsHealthPage } from '@/pages/ops_health_page';
import { OpsPage } from '@/pages/ops_page';
import { OpsReconPage } from '@/pages/ops_recon_page';
import { OpsRumPage } from '@/pages/ops_rum_page';
import { OpsShardsPage } from '@/pages/ops_shards_page';
import { SettingsPage } from '@/pages/settings_page';
import { TeamPage } from '@/pages/team_page';
import { RouteErrorPage } from '@/pages/route_error_page';

function ProtectedLayout() {
  const { bootstrapComplete, licenseNeedsSetup, loading: metaLoading } = useMeta();
  const { authenticated, forbidden, loading: sessionLoading } = useSession();

  if (metaLoading || sessionLoading) {
    return <PageSkeleton />;
  }

  if (!bootstrapComplete) {
    return <Navigate replace to="/activate" />;
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
        <Route element={<Navigate replace to="/settings" />} path="/licence" />
        <Route element={<Navigate replace to="/settings" />} path="/license" />
        <Route element={<Navigate replace to="/settings" />} path="/settings/licence" />
        <Route element={<ProtectedLayout />} errorElement={<RouteErrorPage layout="embedded" />}>
          <Route element={<HomeRedirect />} index />
          <Route element={<CustomersPage />} path="customers" />
          <Route element={<CustomerDetailPage />} path="customers/:id" />
          <Route element={<CampaignsPage />} path="campaigns" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="campaigns/new" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="campaigns/migrate" />
          <Route element={<CampaignEditorPage />} path="campaigns/:id/edit" />
          <Route element={<CampaignIdRedirect />} path="campaigns/:id" />
          <Route element={<InvoiceDetailPage />} path="billing/invoices/:id" />
          <Route element={<Navigate replace to="/exports" />} path="billing" />
          <Route element={<OpsPage />} path="ops" />
          <Route element={<OpsHealthPage />} path="ops/health" />
          <Route element={<PreserveSearchRedirect to="/ops/health" />} path="ops/doctor" />
          <Route element={<OpsSyncErrorsPage />} path="ops/sync-errors" />
          <Route element={<Navigate replace to="/ops/sync-errors" />} path="ops/dlq" />
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
          <Route element={<Navigate replace to="/settings" />} path="settings/license" />
          <Route element={<TeamPage />} path="team" />
          <Route element={<AuditPage />} path="audit" />
          <Route element={<ExportsPage />} path="exports" />
          <Route element={<ReportJobsRoute />} path="reports/jobs" />
          <Route element={<BillingExportsRoute />} path="billing/exports" />
          <Route element={<IntegrationsHubPage />} path="integrations" />
          <Route element={<IntegrationsCostSyncPage />} path="integrations/cost-sync" />
          <Route element={<IntegrationsApiKeysPage />} path="integrations/api-keys" />
          <Route element={<IntegrationsPostbacksPage />} path="integrations/postbacks" />
          <Route element={<IntegrationsDebuggerPage />} path="integrations/debugger" />
          <Route element={<IntegrationsSchemasPage />} path="integrations/schemas" />
          <Route
            element={<IntegrationsPlatformCampaignsPage />}
            path="integrations/platform-campaigns"
          />
          <Route
            element={<IntegrationsAffiliatePresetsPage />}
            path="integrations/affiliate-presets"
          />
          <Route element={<PreserveSearchRedirect to="/exports" />} path="integrations/automation" />
          <Route element={<PreserveSearchRedirect to="/exports" />} path="integrations/smart-alerts" />
          <Route element={<PreserveSearchRedirect to="/exports" />} path="integrations/margin-guard" />
          <Route
            element={<PreserveSearchRedirect to="/exports" />}
            path="integrations/traffic-optimizer"
          />
          <Route
            element={<PreserveSearchRedirect to="/integrations/platform-campaigns" />}
            path="platform-campaigns"
          />
          <Route
            element={<PreserveSearchRedirect to="/integrations/platform-campaigns" />}
            path="platform-campaigns/*"
          />
          <Route element={<ReportIndexLegacyRedirect />} path="reports" />
          <Route element={<ReportLegacyRedirect />} path="reports/*" />
          <Route element={<Navigate replace to="/exports" />} path="dashboards/*" />
          <Route element={<PreserveSearchRedirect to="/exports" />} path="rtb" />
          <Route element={<Navigate replace to="/exports" />} path="rtb/*" />
          <Route element={<Navigate replace to="/exports" />} path="fraud/*" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="creative" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="flows/*" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="landers/*" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="offers" />
          <Route element={<PreserveSearchRedirect to="/campaigns" />} path="offers/*" />
          <Route element={<Navigate replace to="/campaigns" />} path="brands" />
          <Route element={<Navigate replace to="/campaigns" />} path="brand-creatives/*" />
          <Route element={<Navigate replace to="/integrations" />} path="supply/*" />
          <Route element={<Navigate replace to="/ops/domains" />} path="domains" />
          <Route element={<Navigate replace to="/exports" />} path="automation/*" />
          <Route element={<Navigate replace to="/exports" />} path="traffic-optimizer/*" />
          <Route element={<Navigate replace to="/exports" />} path="smart-alerts/*" />
          <Route element={<Navigate replace to="/exports" />} path="margin-guard/*" />
          <Route element={<Navigate replace to="/exports" />} path="portals" />
          <Route element={<Navigate replace to="/exports" />} path="selfserve" />
          <Route element={<Navigate replace to="/exports" />} path="publisher/*" />
          <Route element={<Navigate replace to="/exports" />} path="telegram/*" />
          <Route element={<Navigate replace to="/exports" />} path="report-schedules" />
          <Route element={<Navigate replace to="/exports" />} path="views" />
          <Route element={<Navigate replace to="/exports" />} path="forecast/*" />
          <Route element={<Navigate replace to="/exports" />} path="docs/*" />
          <Route element={<Navigate replace to="/customers" />} path="disputes" />
          <Route element={<Navigate replace to="/settings" />} path="support/*" />
          <Route element={<NotFoundPage />} path="*" />
        </Route>
        <Route element={<NotFoundPage />} path="*" />
      </Route>
    </Routes>
  );
}
