import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router-dom';

import { AppRoutes } from '@/app_routes';
import { MetaProvider } from '@/providers/meta_provider';
import { SessionProvider } from '@/providers/session_provider';
import { ThemeProvider } from '@/providers/theme_provider';
import '@/styles/globals.css';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(
    <BrowserRouter>
      <ThemeProvider>
        <MetaProvider>
          <SessionProvider>
            <AppRoutes />
          </SessionProvider>
        </MetaProvider>
      </ThemeProvider>
    </BrowserRouter>,
  );
}
