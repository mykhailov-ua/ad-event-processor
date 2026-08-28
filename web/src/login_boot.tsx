import { useEffect } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import * as storage from './helpers/storage.js';
import { LoginPage } from './pages/login_page.js';

export function LoginBoot() {
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', storage.getTheme());
  }, []);

  return (
    <BrowserRouter>
      <div id="login-outlet" className="login-root">
        <Routes>
          <Route path="/login" element={<LoginPage />} />
        </Routes>
      </div>
    </BrowserRouter>
  );
}
