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
import { renderLicenseBanner, type LicenseInfo } from './license_banner.js';
import { renderBootstrapBanner } from './bootstrap_banner.js';
import { renderVersionBanner } from './version_banner.js';
import { renderIcon } from './icon.js';
import { installShellStatus } from './shell_status.js';

export type ShellOpts = {
  outlet: HTMLElement;
  extraBanners?: Array<HTMLElement | null | undefined>;
};

export type ShellHandle = {
  node: HTMLElement;
  searchAnchor: HTMLElement;
  setOpsOutboxPending: (count: number) => void;
  destroy: () => void;
};

type MetaPayload = {
  version?: string;
  license?: LicenseInfo;
  bootstrap_complete?: boolean;
};

/**
 * Mount the application shell with sidebar, banners, and main outlet.
 */
export function mountShell(opts: ShellOpts): ShellHandle {
  const extraBanners = (opts.extraBanners ?? []).filter((n): n is HTMLElement => Boolean(n));
  const shell = el('div', { className: 'shell' });
  let sidebarOpen = false;
  let opsOutboxPending = 0;

  const hamburgerRef: { current: HTMLButtonElement | null } = { current: null };
  const sidebarRef: { current: HTMLElement | null } = { current: null };

  let version: string | null = null;
  let license: LicenseInfo | null = null;
  let bootstrapComplete = true;

  let sidebarCollapsed = storage.getSidebarCollapsed();
  let sidebarWidth = storage.getSidebarWidth();

  const overlay = el('div', {
    className: 'drawer-overlay',
    style: { zIndex: '199', display: 'none' },
    'aria-hidden': 'true',
    onClick: () => closeSidebar(),
  });

  const sidebar = el('nav', { className: 'sidebar' });
  sidebarRef.current = sidebar;

  const sidebarHead = el('div', { className: 'sidebar__head' });
  const sidebarBody = el('div', { className: 'sidebar__body' });
  const searchAnchor = el('div');

  function applySidebarWidth(px: number): void {
    if (sidebarCollapsed) return;
    sidebarWidth = clampSidebarWidth(px);
    const w = `${sidebarWidth}px`;
    sidebar.style.width = w;
    sidebar.style.minWidth = w;
    sidebar.style.maxWidth = w;
    document.documentElement.style.setProperty('--sidebar-width', w);
  }

  let lastScrollRatio = 0;

  function applyCollapsed(collapsed: boolean): void {
    const currentMax = sidebarBody.scrollHeight - sidebarBody.clientHeight;
    if (currentMax > 0) {
      lastScrollRatio = sidebarBody.scrollTop / currentMax;
    }
    sidebarCollapsed = collapsed;
    storage.setSidebarCollapsed(collapsed);
    sidebar.classList.add('sidebar--no-transition');
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
    const restoreScroll = (): void => {
      const newMax = sidebarBody.scrollHeight - sidebarBody.clientHeight;
      if (newMax > 0) {
        sidebarBody.scrollTop = Math.round(lastScrollRatio * newMax);
      }
    };
    requestAnimationFrame(restoreScroll);
    const onTransitionEnd = (e: TransitionEvent): void => {
      if (e.target !== sidebar || e.propertyName !== 'width') return;
      sidebar.classList.remove('sidebar--no-transition');
      sidebar.removeEventListener('transitionend', onTransitionEnd);
      restoreScroll();
    };
    sidebar.addEventListener('transitionend', onTransitionEnd);
    window.setTimeout(() => {
      if (sidebar.classList.contains('sidebar--no-transition')) {
        sidebar.classList.remove('sidebar--no-transition');
        restoreScroll();
      }
    }, 200);
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

  function syncResizerAria(): void {
    const bounds = getSidebarWidthBounds();
    resizer.setAttribute('aria-valuemin', String(bounds.min));
    resizer.setAttribute('aria-valuemax', String(bounds.max));
    resizer.setAttribute('aria-valuenow', String(sidebarWidth));
  }

  function onResizerKey(e: KeyboardEvent): void {
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

  resizer.addEventListener('dblclick', (e: MouseEvent) => {
    e.preventDefault();
    if (sidebarCollapsed) return;
    applySidebarWidth(SIDEBAR_WIDTH_DEFAULT);
    storage.setSidebarWidth(sidebarWidth);
    syncResizerAria();
  });

  resizer.addEventListener('pointerdown', (e: PointerEvent) => {
    if (sidebarCollapsed || e.button !== 0) return;
    e.preventDefault();
    resizer.classList.add('sidebar__resizer--active');
    shell.classList.add('shell--sidebar-resizing');
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    function onMove(ev: PointerEvent): void {
      applySidebarWidth(startWidth + (ev.clientX - startX));
      syncResizerAria();
    }

    function onUp(): void {
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

  function closeSidebar(): void {
    sidebarOpen = false;
    overlay.style.display = 'none';
    overlay.setAttribute('aria-hidden', 'true');
    sidebar.classList.remove('sidebar--open');
    hamburger.setAttribute('aria-expanded', 'false');
    if (hamburgerRef.current) hamburgerRef.current.focus();
  }

  function openSidebar(): void {
    sidebarOpen = true;
    overlay.style.display = 'block';
    overlay.setAttribute('aria-hidden', 'false');
    sidebar.classList.add('sidebar--open');
    hamburger.setAttribute('aria-expanded', 'true');
    const firstLink = sidebar.querySelector('.sidebar__link');
    if (firstLink instanceof HTMLElement) firstLink.focus();
  }

  function onSidebarKey(e: KeyboardEvent): void {
    if (!sidebarOpen) return;
    if (e.key === 'Escape') closeSidebar();
  }

  document.addEventListener('keydown', onSidebarKey);

  function onWindowResize(): void {
    if (!sidebarCollapsed) {
      applySidebarWidth(sidebarWidth);
      storage.setSidebarWidth(sidebarWidth);
      syncResizerAria();
    }
  }

  window.addEventListener('resize', onWindowResize);

  function toggleTheme(): void {
    const current = storage.getTheme();
    storage.setTheme(current === 'dark' ? 'light' : 'dark');
    renderFooter();
  }

  function toggleCollapsed(): void {
    applyCollapsed(!sidebarCollapsed);
  }

  async function handleLogout(): Promise<void> {
    const [, err] = await to(apiConfirmed('/api/v1/auth/logout', { method: 'POST' }));
    if (err instanceof ConfirmCancelledError) return;
    auth.logoutLocal();
    window.location.assign('/login');
  }

  function renderNav(): void {
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

  function renderFooter(): void {
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
          onClick: () => { void handleLogout(); },
          title: 'Logout',
        },
          renderIcon('log-out', { size: 15 }),
          el('span', { className: 'sidebar__action-label' }, 'Logout'),
        ),
      ),
    );
  }

  const bannerSlot = el('div', { className: 'banner-slot' });
  const shellStatus = installShellStatus(bannerSlot);

  function renderBanners(): void {
    shellStatus.prependTo(bannerSlot, [
      ...extraBanners,
      renderVersionBanner({ serverVersion: version }),
      renderBootstrapBanner({ bootstrapComplete }),
      renderLicenseBanner({ license: license ?? {} }),
    ]);
  }

  sidebarHead.appendChild(
    el('a', { href: '/', className: 'sidebar__logo', title: 'BidShard home' },
      renderIcon('layers', { size: 32, className: 'sidebar__logo-icon', strokeWidth: 1.5 }),
      el('span', { className: 'sidebar__logo-text' },
        el('span', { className: 'sidebar__logo-bid' }, 'Bid'),
        el('span', { className: 'sidebar__logo-shard' }, 'Shard'),
      ),
    ),
  );
  sidebarHead.appendChild(searchAnchor);
  sidebarBody.appendChild(el('div', { className: 'sidebar__nav' }));
  const sidebarFooter = el('div', { className: 'sidebar__footer' });
  sidebar.appendChild(sidebarHead);
  sidebar.appendChild(sidebarBody);
  sidebar.appendChild(sidebarFooter);
  sidebar.appendChild(resizer);

  const hamburger = el('button', {
    type: 'button',
    className: 'hamburger',
    'aria-label': 'Menu',
    'aria-expanded': 'false',
    onClick: openSidebar,
  }, '☰') as HTMLButtonElement;
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
  renderFooter();
  renderBanners();
  syncResizerAria();

  api('/api/v1/meta')
    .then((res) => {
      const data = res?.data as MetaPayload | null | undefined;
      if (data?.version) version = data.version;
      if (data?.license) license = data.license;
      if (typeof data?.bootstrap_complete === 'boolean') {
        bootstrapComplete = data.bootstrap_complete;
      }
      renderFooter();
      renderBanners();
    })
    .catch(() => {});

  const onRoute = (): void => {
    renderNav();
    hamburger.setAttribute('aria-expanded', sidebarOpen ? 'true' : 'false');
  };
  window.addEventListener('popstate', onRoute);
  window.addEventListener('routechange', onRoute);

  return {
    node: shell,
    searchAnchor,
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
