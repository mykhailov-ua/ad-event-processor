import { lazy, Suspense, type ReactNode } from 'react';
import { Navigate, Route } from 'react-router-dom';
import { REPORT_ROUTE_PATHS } from './models/report.js';
import { ReportRunnerPage } from './pages/report_runner_page.js';

const ReportHubPage = lazy(() =>
  import('./pages/report_hub_page.js').then((mod) => ({
    default: mod.ReportHubPage,
  }))
);

const ReportStubCatchPage = lazy(() =>
  import('./pages/report_stub_catch_page.js').then((mod) => ({
    default: mod.ReportStubCatchPage,
  }))
);

function RouteFallback() {
  return <span className="text-muted">Loading...</span>;
}

function withSuspense(element: ReactNode) {
  return <Suspense fallback={<RouteFallback />}>{element}</Suspense>;
}

const LIVE_REPORT_ROUTES: Array<{ key: string; path: string }> = Object.entries(
  REPORT_ROUTE_PATHS
).map(([key, path]) => ({ key, path }));

export const REPORT_LIVE_ROUTE_PATH_MANIFEST = [
  '/reports',
  ...LIVE_REPORT_ROUTES.map((entry) => entry.path),
  '/reports/ghost-impression-funnel',
] as const;

export const ReportRouteElements = (
  <>
    <Route
      path="/reports"
      element={withSuspense(<ReportHubPage />)}
    />
    <Route
      path="/reports/ghost-impression-funnel"
      element={<Navigate to="/reports/silent-reject-impression-funnel" replace />}
    />
    {LIVE_REPORT_ROUTES.map(({ key, path }) => (
      <Route
        key={key}
        path={path}
        element={withSuspense(<ReportRunnerPage reportKey={key} />)}
      />
    ))}
    <Route
      path="/reports/:reportKey"
      element={withSuspense(<ReportStubCatchPage />)}
    />
  </>
);
