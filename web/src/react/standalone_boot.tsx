import { useEffect } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import * as storage from '../helpers/storage.js';
import { installConfirmHost } from '../ui/confirm_host.js';
import { installCustomScrollbars } from '../ui/custom_scrollbars.js';
import { installToastStack } from '../ui/toast_stack.js';
import { installErrorSurface } from '../lib/error_surface.js';
import { BootstrapPage } from './pages/bootstrap_page.js';
import { InstallDonePage } from './pages/install_done_page.js';
import { NotFoundPage } from './pages/not_found_page.js';

/**
 * Bootstrap / install-done standalone boot (no app shell).
 */
export function StandaloneBoot() {
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', storage.getTheme());
    const root = document.getElementById('root');
    if (!root) return;
    installErrorSurface(root);
    installConfirmHost(root);
    installToastStack(root);
    installCustomScrollbars();
  }, []);

  return (
    <BrowserRouter>
      <div id="login-outlet" className="login-root">
        <Routes>
          <Route path="/bootstrap" element={<BootstrapPage />} />
          <Route path="/install/done" element={<InstallDonePage />} />
          <Route path="*" element={<NotFoundPage />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}
