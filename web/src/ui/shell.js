import { el } from '../lib/dom.js';
import { api } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import { to } from '../lib/to.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import {
  clampSidebarWidth,
  getSidebarWidthBounds,
  SIDEBAR_COLLAPSED_WIDTH,
  SIDEBAR_WIDTH_DEFAULT,
} from '../helpers/sidebar_layout.js';
import { visibleNavGroups } from '../helpers/nav_config.js';
import { renderLicenseBanner } from './license_banner.js';
import { renderBootstrapBanner } from './bootstrap_banner.js';
import { renderVersionBanner } from './version_banner.js';
import { renderIcon } from './icon.js';
import { installShellStatus } from './shell_status.js';

/**
 * Mount the application shell with sidebar, banners, and main outlet.
 *
 * @param {{
 *   outlet: HTMLElement,
 *   extraBanners?: Array<HTMLElement|null|undefined>,
 *   onOpenSearch?: (query?: string) => void,
 * }} opts
 * @returns {{ node: HTMLElement, destroy: () => void }}
 */
export function mountShell(opts) {
  const extraBanners = (opts.extraBanners ?? []).filter(Boolean);
  const onOpenSearch = opts.onOpenSearch ?? (() => {});
  const shell = el('div', { className: 'shell' });
  let sidebarOpen = false;
  let opsOutboxPending = 0;

  const hamburgerRef = { current: null };
  const sidebarRef = { current: null };

  let version = null;
  let license = null;
  let bootstrapComplete = true;

  let sidebarCollapsed = storage.getSidebarCollapsed();
  let sidebarWidth = storage.getSidebarWidth();

  const overlay = el('div', {
    className: 'drawer-overlay',
    style: { zIndex: 199, display: 'none' },
    'aria-hidden': 'true',
    onClick: () => closeSidebar(),
  });

  const sidebar = el('nav', { className: 'sidebar' });
  sidebarRef.current = sidebar;

  const sidebarScroll = el('div', { className: 'sidebar__scroll' });

  function applySidebarWidth(px) {
    if (sidebarCollapsed) return;
    sidebarWidth = clampSidebarWidth(px);
    const w = `${sidebarWidth}px`;
    sidebar.style.width = w;
    sidebar.style.minWidth = w;
    sidebar.style.maxWidth = w;
    document.documentElement.style.setProperty('--sidebar-width', w);
  }

  let lastScrollRatio = 0;

  function applyCollapsed(collapsed) {
    const currentMax = sidebarScroll.scrollHeight - sidebarScroll.clientHeight;
    if (currentMax > 0) {
      lastScrollRatio = sidebarScroll.scrollTop / currentMax;
    }
    sidebarCollapsed = collapsed;
    storage.setSidebarCollapsed(collapsed);
    sidebar.classList.toggle('sidebar--collapsed', collapsed);
    if (collapsed) {
      const w = `${SIDEBAR_COLLAPSED_WIDTH}px`;
      document.documentElement.style.setProperty('--sidebar-width', w);
      sidebar.style.width = w;
      sidebar.style.minWidth = w;
      sidebar.style.maxWidth = w;
    } else {
      applySidebarWidth(sidebarWidth);
    }
    renderFooter();
    requestAnimationFrame(() => {
      const newMax = sidebarScroll.scrollHeight - sidebarScroll.clientHeight;
      if (newMax > 0) {
        sidebarScroll.scrollTop = Math.round(lastScrollRatio * newMax);
      }
    });
  }

  if (sidebarCollapsed) {
    applyCollapsed(true);
  } else {
    applySidebarWidth(sidebarWidth);
  }

  const resizer = el('div', {
    className: 'sidebar__resizer',
    role: 'separator',
    'aria-orientation': 'vertical',
    'aria-label': 'Resize sidebar',
    'aria-valuemin': String(getSidebarWidthBounds().min),
    'aria-valuemax': String(getSidebarWidthBounds().max),
    'aria-valuenow': String(sidebarWidth),
    tabIndex: 0,
  });

  function syncResizerAria() {
    const bounds = getSidebarWidthBounds();
    resizer.setAttribute('aria-valuemin', String(bounds.min));
    resizer.setAttribute('aria-valuemax', String(bounds.max));
    resizer.setAttribute('aria-valuenow', String(sidebarWidth));
  }

  function onResizerKey(e) {
    if (sidebarCollapsed) return;
    if (e.key === 'ArrowLeft') {
      e.preventDefault();
      applySidebarWidth(sidebarWidth - 8);
      storage.setSidebarWidth(sidebarWidth);
      syncResizerAria();
    } else if (e.key === 'ArrowRight') {
      e.preventDefault();
      applySidebarWidth(sidebarWidth + 8);
      storage.setSidebarWidth(sidebarWidth);
      syncResizerAria();
    }
  }

  resizer.addEventListener('keydown', onResizerKey);

  resizer.addEventListener('dblclick', (e) => {
    e.preventDefault();
    if (sidebarCollapsed) return;
    applySidebarWidth(SIDEBAR_WIDTH_DEFAULT);
    storage.setSidebarWidth(sidebarWidth);
    syncResizerAria();
  });

  resizer.addEventListener('pointerdown', (e) => {
    if (sidebarCollapsed || e.button !== 0) return;
    e.preventDefault();
    resizer.classList.add('sidebar__resizer--active');
    shell.classList.add('shell--sidebar-resizing');
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    function onMove(ev) {
      applySidebarWidth(startWidth + (ev.clientX - startX));
      syncResizerAria();
    }

    function onUp() {
      resizer.classList.remove('sidebar__resizer--active');
      shell.classList.remove('shell--sidebar-resizing');
      storage.setSidebarWidth(sidebarWidth);
      document.removeEventListener('pointermove', onMove);
      document.removeEventListener('pointerup', onUp);
    }

    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
  });

  const mainContent = el('div', { className: 'main__content' });
  const outletWrap = opts.outlet;

  function closeSidebar() {
    sidebarOpen = false;
    overlay.style.display = 'none';
    overlay.setAttribute('aria-hidden', 'true');
    sidebar.classList.remove('sidebar--open');
    hamburger.setAttribute('aria-expanded', 'false');
    if (hamburgerRef.current) hamburgerRef.current.focus();
  }

  function openSidebar() {
    sidebarOpen = true;
    overlay.style.display = 'block';
    overlay.setAttribute('aria-hidden', 'false');
    sidebar.classList.add('sidebar--open');
    hamburger.setAttribute('aria-expanded', 'true');
    const firstLink = sidebar.querySelector('.sidebar__link');
    if (firstLink instanceof HTMLElement) firstLink.focus();
  }

  function onSidebarKey(e) {
    if (!sidebarOpen) return;
    if (e.key === 'Escape') closeSidebar();
  }

  document.addEventListener('keydown', onSidebarKey);

  function onWindowResize() {
    if (!sidebarCollapsed) {
      applySidebarWidth(sidebarWidth);
      storage.setSidebarWidth(sidebarWidth);
      syncResizerAria();
    }
  }

  window.addEventListener('resize', onWindowResize);

  function toggleTheme() {
    const current = storage.getTheme();
    storage.setTheme(current === 'dark' ? 'light' : 'dark');
    renderFooter();
  }

  function toggleCollapsed() {
    applyCollapsed(!sidebarCollapsed);
  }

  async function handleLogout() {
    const [, err] = await to(apiConfirmed('/api/v1/auth/logout', { method: 'POST' }));
    if (err instanceof ConfirmCancelledError) return;
    auth.logoutLocal();
    window.location.assign('/login');
  }

  function renderNav() {
    const user = auth.getUser();
    const permissions = user?.permissions ?? [];
    const groups = visibleNavGroups(permissions);
    const navContainer = sidebar.querySelector('.sidebar__nav');
    if (!navContainer) return;
    navContainer.replaceChildren();
    for (const group of groups) {
      const groupEl = el('div', { className: 'sidebar__group' },
        el('div', { className: 'sidebar__group-label' }, group.title),
      );
      for (const link of group.links) {
        const path = window.location.pathname;
        const isActive = link.to === '/' ? path === '/' : path === link.to || path.startsWith(link.to + '/');
        groupEl.appendChild(
          el('a', {
            href: link.to,
            className: 'sidebar__link' + (isActive ? ' sidebar__link--active' : ''),
            title: link.label,
            'aria-label': link.label,
            onClick: () => closeSidebar(),
          },
            link.icon ? renderIcon(link.icon, { size: 18, className: 'sidebar__link-icon', strokeWidth: 1.5 }) : null,
            el('span', { className: 'sidebar__link-label' }, link.label),
            link.to === '/ops' && opsOutboxPending > 0
              ? el('span', {
                className: 'sidebar__badge',
                'aria-label': `${opsOutboxPending} outbox pending`,
              }, String(opsOutboxPending))
              : null,
          ),
        );
      }
      navContainer.appendChild(groupEl);
    }
  }

  function renderFooter() {
    const footer = sidebar.querySelector('.sidebar__footer');
    if (!footer) return;
    const user = auth.getUser();
    const theme = storage.getTheme();
    const themeIcon = theme === 'dark' ? 'sun' : 'moon';
    const collapseIcon = sidebarCollapsed ? 'panel-left-open' : 'panel-left-close';

    footer.replaceChildren(
      el('div', { className: 'sidebar__account' },
        user?.email
          ? el('span', { className: 'sidebar__email', title: user.email }, user.email)
          : null,
        version
          ? el('span', { className: 'sidebar__version' }, `v${version}`)
          : null,
      ),
      el('div', { className: 'sidebar__actions' },
        el('button', {
          type: 'button',
          className: 'sidebar__action-btn',
          onClick: toggleCollapsed,
          title: sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar',
          'aria-expanded': sidebarCollapsed ? 'false' : 'true',
        },
          renderIcon(collapseIcon, { size: 15 }),
          el('span', { className: 'sidebar__action-label' }, sidebarCollapsed ? 'Expand' : 'Collapse'),
        ),
        el('button', {
          type: 'button',
          className: 'sidebar__action-btn',
          onClick: toggleTheme,
          title: 'Toggle color theme',
        },
          renderIcon(themeIcon, { size: 15 }),
          el('span', { className: 'sidebar__action-label' }, theme === 'dark' ? 'Light' : 'Dark'),
        ),
        el('button', {
          type: 'button',
          className: 'sidebar__action-btn sidebar__action-btn--muted',
          onClick: handleLogout,
          title: 'Logout',
        },
          renderIcon('log-out', { size: 15 }),
          el('span', { className: 'sidebar__action-label' }, 'Logout'),
        ),
      ),
    );
  }

  function renderSearch() {
    const searchWrap = sidebar.querySelector('.sidebar__search');
    if (!searchWrap) return;
    const input = el('input', {
      type: 'search',
      className: 'sidebar__search-input',
      placeholder: 'Search pages…',
      'aria-label': 'Search pages',
      onFocus: (e) => {
        const q = e.target.value;
        e.target.blur();
        onOpenSearch(q);
      },
    });
    searchWrap.replaceChildren(
      renderIcon('search', { size: 16, className: 'sidebar__search-icon' }),
      input,
    );
  }

  const bannerSlot = el('div', { className: 'banner-slot' });
  const shellStatus = installShellStatus(bannerSlot);

  function renderBanners() {
    shellStatus.prependTo(bannerSlot, [
      ...extraBanners,
      renderVersionBanner({ serverVersion: version }),
      renderBootstrapBanner({ bootstrapComplete }),
      renderLicenseBanner({ license }),
    ]);
  }

  sidebarScroll.appendChild(
    el('a', { href: '/', className: 'sidebar__logo', title: 'BidShard home' },
      renderIcon('layers', { size: 32, className: 'sidebar__logo-icon', strokeWidth: 1.5 }),
      el('span', { className: 'sidebar__logo-text' },
        el('span', { className: 'sidebar__logo-bid' }, 'Bid'),
        el('span', { className: 'sidebar__logo-shard' }, 'Shard'),
      ),
    ),
  );
  sidebarScroll.appendChild(el('div', { className: 'sidebar__search' }));
  sidebarScroll.appendChild(el('div', { className: 'sidebar__nav' }));
  sidebarScroll.appendChild(el('div', { className: 'sidebar__footer' }));
  sidebar.appendChild(sidebarScroll);
  sidebar.appendChild(resizer);

  const hamburger = el('button', {
    type: 'button',
    className: 'hamburger',
    'aria-label': 'Menu',
    'aria-expanded': 'false',
    onClick: openSidebar,
  }, '☰');
  hamburgerRef.current = hamburger;

  mainContent.appendChild(bannerSlot);
  mainContent.appendChild(outletWrap);

  shell.appendChild(overlay);
  shell.appendChild(sidebar);
  shell.appendChild(el('main', { className: 'main' },
    el('div', { className: 'main__toolbar' }, hamburger),
    mainContent,
  ));

  renderNav();
  renderSearch();
  renderFooter();
  renderBanners();
  syncResizerAria();

  api('/api/v1/meta')
    .then((res) => {
      const data = res?.data;
      if (data?.version) version = data.version;
      if (data?.license) license = data.license;
      if (typeof data?.bootstrap_complete === 'boolean') {
        bootstrapComplete = data.bootstrap_complete;
      }
      renderFooter();
      renderBanners();
    })
    .catch(() => {});

  const onRoute = () => {
    renderNav();
    hamburger.setAttribute('aria-expanded', sidebarOpen ? 'true' : 'false');
  };
  window.addEventListener('popstate', onRoute);
  window.addEventListener('routechange', onRoute);

  return {
    node: shell,
    setOpsOutboxPending(count) {
      const next = Math.max(0, Number(count) || 0);
      if (next === opsOutboxPending) return;
      opsOutboxPending = next;
      renderNav();
    },
    destroy() {
      document.removeEventListener('keydown', onSidebarKey);
      window.removeEventListener('popstate', onRoute);
      window.removeEventListener('routechange', onRoute);
      window.removeEventListener('resize', onWindowResize);
      shellStatus.destroy();
    },
  };
}
