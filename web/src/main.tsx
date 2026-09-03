import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

import { AppRoutes } from '@/app_routes';
import { AppErrorBoundary } from '@/shell/app_error_boundary';
import { MetaProvider } from '@/providers/meta_provider';
import { SessionProvider } from '@/providers/session_provider';
import { ThemeProvider } from '@/providers/theme_provider';
import { initAdminDevModeFromUrl } from '@/lib/admin_dev_mode';
import '@/styles/app.css';

initAdminDevModeFromUrl();

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(
    <AppErrorBoundary layout="standalone">
      <BrowserRouter>
        <ThemeProvider>
          <MetaProvider>
            <SessionProvider>
              <AppRoutes />
            </SessionProvider>
          </MetaProvider>
        </ThemeProvider>
      </BrowserRouter>
    </AppErrorBoundary>,
  );
}
