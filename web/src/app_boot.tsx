import { lazy, Suspense, useEffect, useState } from 'react';
import * as storage from './helpers/storage.js';
import * as auth from './helpers/auth.js';
import { api } from './helpers/api_client.js';
import { redirectToLogin } from './helpers/session.js';
import { syncDevModeAttribute } from './helpers/dev_mode.js';
import { to } from './lib/to.js';

const AppProviders = lazy(() =>
  import('./ui/shell/app_providers.js').then((mod) => ({
    default: mod.AppProviders,
  }))
);
const EulaGate = lazy(() =>
  import('./ui/shell/eula_gate.js').then((mod) => ({
    default: mod.EulaGate,
  }))
);
const AppShell = lazy(() => import('./app_shell.js').then((mod) => ({ default: mod.AppShell })));

type EulaPayload = {
  version: string;
  text: string;
};

type BootResult = { ok: false } | { ok: true; eula?: EulaPayload };

async function prepareAuthenticatedApp(): Promise<BootResult> {
  document.documentElement.setAttribute('data-theme', storage.getTheme());
  syncDevModeAttribute();

  auth.hydrateFromBoot();

  const [meRes, meErr] = await to(api('/api/v1/auth/me'));
  if (meErr) {
    redirectToLogin('session');
    return { ok: false };
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
      return {
        ok: true,
        eula: {
          version: eula.version ?? '',
          text: eula.text ?? '',
        },
      };
    }
  }

  return { ok: true };
}

export function AppBoot() {
  const [phase, setPhase] = useState<'loading' | 'eula' | 'ready'>('loading');
  const [eula, setEula] = useState<EulaPayload | null>(null);

  useEffect(() => {
    void prepareAuthenticatedApp().then((result) => {
      if (!result.ok) return;
      if (result.eula) {
        setEula(result.eula);
        setPhase('eula');
        return;
      }
      setPhase('ready');
    });
  }, []);

  if (phase === 'loading') {
    return (
      <div className="login-root">
        <span className="text-muted">Loading...</span>
      </div>
    );
  }

  if (phase === 'eula' && eula) {
    return (
      <Suspense fallback={<span className="text-muted">Loading...</span>}>
        <AppProviders>
          <EulaGate version={eula.version} text={eula.text} onAccepted={() => setPhase('ready')} />
        </AppProviders>
      </Suspense>
    );
  }

  return (
    <Suspense
      fallback={
        <div className="login-root">
          <span className="text-muted">Loading...</span>
        </div>
      }
    >
      <AppProviders>
        <AppShell />
      </AppProviders>
    </Suspense>
  );
}
