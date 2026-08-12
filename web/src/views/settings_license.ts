import type { ViewHandle } from '../lib/router_types.js';
import type { LicenseStatusDTO } from '../types/api/license.js';
import { el, replaceChildren } from '../lib/dom.js';
import { createResource, type ResourceState } from '../lib/fetch_resource.js';
import { renderErrorBlock } from '../ui/error_block.js';
import { renderBreadcrumbs } from '../ui/breadcrumbs.js';
import { renderStatusBadge } from '../ui/status_badge.js';
import { renderIcon } from '../ui/icon.js';
import { surfaceServiceErrorToast } from '../helpers/service_error_toast.js';
import { renderButtonLink } from '../ui/button.js';

/**
 * Mount license status view with link to apply on platform settings.
 */
export function mount(container: HTMLElement): ViewHandle {
  let destroyed = false;
  const state: ResourceState<LicenseStatusDTO> = { data: null, loading: true, error: null };
  let lastError: unknown = null;

  function render(): void {
    if (destroyed) return;

    replaceChildren(container,
      el('div', { className: 'page-header' },
        renderBreadcrumbs([
          { label: 'Settings', href: '/settings' },
          { label: 'License' },
        ]),
        el('div', { className: 'page-header__row' },
          el('div', { className: 'flex items-center gap-2' },
            renderIcon('key', { size: 20, className: 'text-muted' }),
            el('h1', { className: 'page-header__title' }, 'License'),
          ),
        ),
        el('p', { className: 'text-muted text-sm' },
          'On-prem deployment license. Apply a new JWT from ',
          el('a', { href: '/settings' }, 'Platform settings'),
          '.',
        ),
      ),
      state.loading ? el('p', { className: 'text-muted' }, 'Loading…') : null,
      state.error ? renderErrorBlock(state.error, 'Failed to load license status') : null,
      state.data
        ? el('section', {
          className: 'section-card stack',
          'data-testid': 'license-status-panel',
        },
          el('dl', { className: 'definition-list' },
            el('dt', null, 'Deployment ID'),
            el('dd', { className: 'font-mono' }, state.data.deployment_id ?? '—'),
            el('dt', null, 'State'),
            el('dd', null,
              state.data.state
                ? renderStatusBadge(state.data.state)
                : '—',
            ),
            el('dt', null, 'Valid until'),
            el('dd', null,
              state.data.valid_until
                ? new Date(state.data.valid_until).toLocaleString()
                : '—',
            ),
          ),
          renderButtonLink({
            label: 'Apply license on Platform settings',
            href: '/settings',
            variant: 'secondary',
            size: 'sm',
            testId: 'license-apply-link',
          }),
        )
        : null,
    );
  }

  const resource = createResource<LicenseStatusDTO>(
    () => '/api/v1/license/status',
    {
      onUpdate: (s) => {
        if (s.error !== lastError) {
          lastError = s.error;
          if (s.error) surfaceServiceErrorToast(s.error);
        }
        Object.assign(state, s);
        render();
      },
    },
  );

  render();

  return {
    destroy() {
      destroyed = true;
      resource.destroy();
    },
  };
}
