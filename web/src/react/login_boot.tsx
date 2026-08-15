import { useEffect } from 'react';
import { BrowserRouter, Route, Routes } from 'react-router-dom';
import * as storage from '../helpers/storage.js';
import { installCustomScrollbars } from '../ui/custom_scrollbars.js';
import { installToastStack } from '../ui/toast_stack.js';
import { LoginPage } from './pages/login_page.js';

/**
 * Login entry boot — theme + toast, no authenticated shell.
 */
export function LoginBoot() {
  useEffect(() => {
    document.documentElement.setAttribute('data-theme', storage.getTheme());
    installToastStack(document.getElementById('root') ?? document.body);
    installCustomScrollbars();
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
