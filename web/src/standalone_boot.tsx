import { lazy, Suspense, useEffect } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import * as storage from './helpers/storage.js';
import { BootstrapPage } from './pages/bootstrap_page.js';
import { InstallDonePage } from './pages/install_done_page.js';
import { NotFoundPage } from './pages/not_found_page.js';

const AppProviders = lazy(() => import('./components/app_providers.js').then((mod) => ({
  default: mod.AppProviders,
})));

/**
 * Bootstrap / install-done standalone boot (no app shell).
 */
export function StandaloneBoot() {
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', storage.getTheme());
  }, []);

  return (
    <Suspense fallback={<span className="text-muted">Loading…</span>}>
      <AppProviders>
        <BrowserRouter>
          <div id="login-outlet" className="login-root">
            <Routes>
              <Route path="/bootstrap" element={<BootstrapPage />} />
              <Route path="/install/done" element={<InstallDonePage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </div>
        </BrowserRouter>
      </AppProviders>
    </Suspense>
  );
}
