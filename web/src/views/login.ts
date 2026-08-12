import { el, replaceChildren, eventTargetValue } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import type { AuthUser } from '../helpers/auth.js';
import * as auth from '../helpers/auth.js';
import type { RouteContext, ViewHandle } from '../lib/router_types.js';
import { renderButton } from '../ui/button.js';

const REASON_MESSAGES: Record<string, string> = {
  session: 'Your session expired. Sign in again.',
};

type LoginState = {
  email: string;
  password: string;
  loading: boolean;
  error: string | null;
};

type LoginResponse = {
  user?: AuthUser;
};

type MetaResponse = {
  bootstrap_complete?: boolean;
};


/**
 * Mount the login form and handle credential submission.
 */
export function mount(container: HTMLElement, ctx: RouteContext): ViewHandle {
  let destroyed = false;
  const reason = ctx.query.get('reason');
  const state: LoginState = {
    email: '',
    password: '',
    loading: false,
    error: null,
  };

  function render(): void {
    if (destroyed) return;
    const children = [
      el('div', { className: 'login-page' },
        el('div', { className: 'login-box' },
          el('h1', { className: 'login-box__title' },
            el('span', { className: 'login-box__title-bid' }, 'Bid'),
            el('span', { className: 'login-box__title-shard' }, 'Shard'),
          ),
          el('p', { className: 'login-box__sub' }, 'Admin Control Plane'),
          (reason && REASON_MESSAGES[reason]) || state.error
            ? el('div', { className: 'login-box__notices' },
              reason && REASON_MESSAGES[reason]
                ? el('div', { className: 'login-box__notice login-box__notice--warning' }, REASON_MESSAGES[reason])
                : null,
              state.error
                ? el('div', { className: 'login-box__notice login-box__notice--error' }, state.error)
                : null,
            )
            : null,
          el('form', { onSubmit: handleSubmit },
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label' }, 'Email'),
              el('input', {
                type: 'email',
                className: 'form-input',
                required: true,
                value: state.email,
                onInput: (e: Event) => { state.email = eventTargetValue(e); },
              }),
            ),
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label' }, 'Password'),
              el('input', {
                type: 'password',
                className: 'form-input',
                required: true,
                value: state.password,
                onInput: (e: Event) => { state.password = eventTargetValue(e); },
              }),
            ),
            el('div', { className: 'form-actions' },
              renderButton({
                label: state.loading ? 'Signing in...' : 'Sign in',
                variant: 'primary',
                type: 'submit',
                className: 'btn--block',
                loading: state.loading,
                disabled: state.loading,
              }),
            ),
          ),
        ),
      ),
    ];
    replaceChildren(container, ...children);
  }

  async function handleSubmit(e: Event): Promise<void> {
    e.preventDefault();
    state.loading = true;
    state.error = null;
    render();
    const [res, err] = await to(api('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email: state.email, password: state.password }),
    }));
    if (err) {
      state.error = err.message || 'Login failed';
    } else if (res) {
      const csrf = res.headers.get('X-CSRF-Token');
      if (csrf) auth.setCsrfFromLoginResponse(csrf);
      const data = res.data as LoginResponse | null;
      if (data?.user) auth.setUser(data.user);
      window.location.assign('/');
    }
    state.loading = false;
    render();
  }

  to(api('/api/v1/meta')).then(([metaRes, metaErr]) => {
    if (destroyed || metaErr || !metaRes) return;
    const data = metaRes.data as MetaResponse | null;
    if (data?.bootstrap_complete === false) {
      window.location.assign('/bootstrap');
    }
  });

  render();

  return {
    destroy() {
      destroyed = true;
    },
  };
}
