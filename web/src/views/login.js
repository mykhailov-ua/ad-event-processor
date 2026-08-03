import { el, replaceChildren } from '../lib/dom.js';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import * as auth from '../helpers/auth.js';

const REASON_MESSAGES = {
  session: 'Your session expired. Sign in again.',
};

/**
 * Mount the login form and handle credential submission.
 *
 * @param {HTMLElement} container
 * @param {{ query: URLSearchParams, navigate: (path: string) => void }} ctx
 * @returns {import('../lib/router.js').ViewHandle}
 */
export function mount(container, ctx) {
  let destroyed = false;
  const reason = ctx.query.get('reason');
  const state = {
    email: '',
    password: '',
    loading: false,
    error: null,
  };

  function render() {
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
                onInput: (e) => { state.email = e.target.value; },
              }),
            ),
            el('div', { className: 'form-field' },
              el('label', { className: 'form-label' }, 'Password'),
              el('input', {
                type: 'password',
                className: 'form-input',
                required: true,
                value: state.password,
                onInput: (e) => { state.password = e.target.value; },
              }),
            ),
            el('div', { className: 'form-actions' },
              el('button', {
                type: 'submit',
                className: 'btn btn--primary btn--block',
                disabled: state.loading,
              }, state.loading ? 'Signing in...' : 'Sign in'),
            ),
          ),
        ),
      ),
    ];
    replaceChildren(container, ...children);
  }

  async function handleSubmit(e) {
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
    } else {
      const csrf = res.headers.get('X-CSRF-Token');
      if (csrf) auth.setCsrfFromLoginResponse(csrf);
      if (res.data?.user) auth.setUser(res.data.user);
      window.location.assign('/');
    }
    state.loading = false;
    render();
  }

  to(api('/api/v1/meta')).then(([metaRes, metaErr]) => {
    if (destroyed || metaErr) return;
    if (metaRes?.data?.bootstrap_complete === false) {
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
