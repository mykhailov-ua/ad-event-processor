import { createRoot } from 'react-dom/client';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { ActivatePage } from '@/pages/activate_page';
import { LoginPage } from '@/pages/login_page';
import { SetupPage } from '@/pages/setup_page';
import { MetaProvider } from '@/providers/meta_provider';
import { ThemeProvider } from '@/providers/theme_provider';
import '@/styles/app.css';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(
    <BrowserRouter>
      <ThemeProvider>
        <MetaProvider>
          <Routes>
            <Route element={<SetupPage />} path="/setup" />
            <Route element={<ActivatePage />} path="/activate" />
            <Route element={<LoginPage />} path="*" />
          </Routes>
        </MetaProvider>
      </ThemeProvider>
    </BrowserRouter>,
  );
}
