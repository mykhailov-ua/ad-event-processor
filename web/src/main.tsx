import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

import { AppRoutes } from '@/app_routes';
import { AppErrorBoundary } from '@/shell/app_error_boundary';
import { MetaProvider } from '@/context/meta_context';
import { SessionProvider } from '@/context/session_context';
import { ThemeProvider } from '@/context/theme_context';
import { initAdminDevModeFromUrl } from '@/lib/admin_dev_mode';
import { initDevMockRoleFromUrl } from '@/lib/dev_mock_role';
import '@/styles/app.css';

initAdminDevModeFromUrl();
initDevMockRoleFromUrl();

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
