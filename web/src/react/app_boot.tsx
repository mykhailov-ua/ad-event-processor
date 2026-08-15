import { lazy, Suspense, useEffect, useState } from 'react';
import * as storage from '../helpers/storage.js';
import * as auth from '../helpers/auth.js';
import { api } from '../helpers/api_client.js';
import { redirectToLogin } from '../helpers/session.js';
import { syncDevModeAttribute } from '../helpers/dev_mode.js';
import { installConfirmHost } from '../ui/confirm_host.js';
import { installCustomScrollbars } from '../ui/custom_scrollbars.js';
import { installToastStack } from '../ui/toast_stack.js';
import { mountEulaGate } from '../ui/eula_gate.js';
import { installErrorSurface } from '../lib/error_surface.js';
import { to } from '../lib/to.js';

const AppShell = lazy(() => import('./app_shell.js').then((mod) => ({ default: mod.AppShell })));

/**
 * Run auth/me + optional EULA gate before rendering the app shell.
 */
async function prepareAuthenticatedApp(root: HTMLElement): Promise<boolean> {
  document.documentElement.setAttribute('data-theme', storage.getTheme());
  syncDevModeAttribute();
  installErrorSurface(root);
  installConfirmHost(root);
  installToastStack(root);
  installCustomScrollbars();

  auth.hydrateFromBoot();

  const [meRes, meErr] = await to(api('/api/v1/auth/me'));
  if (meErr) {
    redirectToLogin('session');
    return false;
  }

  const csrf = meRes.headers.get('X-CSRF-Token');
  if (csrf) auth.setCsrfFromLoginResponse(csrf);

  const me = meRes.data as {
    id: string;
    email: string;
    role: string;
    customer_id?: string;
    permissions?: string[];
  };
  auth.setUser({
    id: me.id,
    email: me.email,
    role: me.role,
    customer_id: me.customer_id ?? '',
    permissions: me.permissions ?? [],
  });

  const [eulaRes, eulaErr] = await to(api('/api/v1/eula'));
  if (!eulaErr && eulaRes?.data) {
    const eula = eulaRes.data as { required?: boolean; version?: string; text?: string };
    if (eula.required) {
      const accepted = await mountEulaGate(root, {
        version: eula.version ?? '',
        text: eula.text ?? '',
      });
      if (!accepted) return false;
    }
  }

  return true;
}

/**
 * Boot gate: loading screen, then React shell after session is ready.
 */
export function AppBoot() {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const root = document.getElementById('root');
    if (!root) return;
    void prepareAuthenticatedApp(root).then((ok) => {
      if (ok) setReady(true);
    });
  }, []);

  if (!ready) {
    return (
      <div
        className="main"
        style={{ alignItems: 'center', justifyContent: 'center' }}
      >
        <span className="text-muted">Loading…</span>
      </div>
    );
  }

  return (
    <Suspense
      fallback={(
        <div
          className="main"
          style={{ alignItems: 'center', justifyContent: 'center' }}
        >
          <span className="text-muted">Loading…</span>
        </div>
      )}
    >
      <AppShell />
    </Suspense>
  );
}
