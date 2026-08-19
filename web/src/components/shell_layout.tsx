import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { to } from '../lib/to.js';
import { api } from '../helpers/api_client.js';
import { apiConfirmed } from '../helpers/confirmed_api.js';
import { ConfirmCancelledError } from '../helpers/confirm_ui.js';
import * as auth from '../helpers/auth.js';
import * as storage from '../helpers/storage.js';
import {
  clampSidebarWidth,
  getSidebarWidthBounds,
  SIDEBAR_COLLAPSED_WIDTH,
  SIDEBAR_WIDTH_DEFAULT,
} from '../helpers/sidebar_layout.js';
import { visibleNavGroups } from '../helpers/nav_config.js';
import { startOpsOutboxBadge } from '../helpers/ops_outbox_badge.js';
import { startRUMCollector } from '../helpers/rum_collector.js';
import { BootstrapBanner } from './bootstrap_banner.js';
import { Icon } from './icon.js';
import { IdempotencyRecoveryBanner } from './idempotency_recovery_banner.js';
import { LicenseBanner, type LicenseInfo } from './license_banner.js';
import { ShellStatusBanners } from './shell_status_banners.js';
import { SidebarSearch, type SidebarSearchHandle } from './sidebar_search.js';
import { VersionBanner } from './version_banner.js';

type MetaPayload = {
  version?: string;
  license?: LicenseInfo;
  support_url?: string;
  bootstrap_complete?: boolean;
};

export type ShellLayoutProps = {
  children: ReactNode;
};

export function ShellLayout({ children }: ShellLayoutProps) {
  const location = useLocation();
  const path = location.pathname;

  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => storage.getSidebarCollapsed());
  const [sidebarWidth, setSidebarWidthState] = useState(() => storage.getSidebarWidth());
  const [opsOutboxPending, setOpsOutboxPending] = useState(0);
  const [version, setVersion] = useState<string | null>(null);
  const [license, setLicense] = useState<LicenseInfo | null>(null);
  const [supportUrl, setSupportUrl] = useState<string | undefined>(undefined);
  const [bootstrapComplete, setBootstrapComplete] = useState(true);
  const [resizing, setResizing] = useState(false);

  const hamburgerRef = useRef<HTMLButtonElement>(null);
  const sidebarBodyRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<SidebarSearchHandle>(null);
  const scrollRatioRef = useRef(0);

  const applySidebarWidth = useCallback(
    (px: number) => {
      const next = clampSidebarWidth(px);
      setSidebarWidthState(next);
      if (!sidebarCollapsed) {
        const w = `${next}px`;
        document.documentElement.style.setProperty('--sidebar-width', w);
      }
      return next;
    },
    [sidebarCollapsed]
  );

  useEffect(() => {
    if (sidebarCollapsed) {
      const w = `${SIDEBAR_COLLAPSED_WIDTH}px`;
      document.documentElement.style.setProperty('--sidebar-width', w);
    } else {
      const w = `${sidebarWidth}px`;
      document.documentElement.style.setProperty('--sidebar-width', w);
    }
  }, [sidebarCollapsed, sidebarWidth]);

  useEffect(() => {
    void api('/api/v1/meta')
      .then((res) => {
        const data = res?.data as MetaPayload | null | undefined;
        if (data?.version) setVersion(data.version);
        if (data?.license) setLicense(data.license);
        if (data?.support_url) setSupportUrl(data.support_url);
        if (typeof data?.bootstrap_complete === 'boolean') {
          setBootstrapComplete(data.bootstrap_complete);
        }
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    const user = auth.getUser();
    const perms = user?.permissions ?? [];
    if (!perms.includes('shards:read')) return undefined;
    const feed = startOpsOutboxBadge(setOpsOutboxPending);
    return () => feed.destroy?.();
  }, []);

  useEffect(() => {
    const rum = startRUMCollector();
    return () => rum.stop();
  }, []);

  useEffect(() => {
    const onGlobalKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        searchRef.current?.focus('');
      }
    };
    document.addEventListener('keydown', onGlobalKey);
    return () => document.removeEventListener('keydown', onGlobalKey);
  }, []);

  useEffect(() => {
    if (!sidebarOpen) return undefined;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') closeSidebar();
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [sidebarOpen]);

  const closeSidebar = () => {
    setSidebarOpen(false);
    hamburgerRef.current?.focus();
  };

  const openSidebar = () => {
    setSidebarOpen(true);
    const firstLink = document.querySelector('.sidebar__link');
    if (firstLink instanceof HTMLElement) firstLink.focus();
  };

  const applyCollapsed = (collapsed: boolean) => {
    const body = sidebarBodyRef.current;
    if (body) {
      const currentMax = body.scrollHeight - body.clientHeight;
      if (currentMax > 0) scrollRatioRef.current = body.scrollTop / currentMax;
    }
    setSidebarCollapsed(collapsed);
    storage.setSidebarCollapsed(collapsed);
    requestAnimationFrame(() => {
      const el = sidebarBodyRef.current;
      if (!el) return;
      const newMax = el.scrollHeight - el.clientHeight;
      if (newMax > 0) el.scrollTop = Math.round(scrollRatioRef.current * newMax);
    });
  };

  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const [theme, setTheme] = useState(() => storage.getTheme());
  const themeIcon = theme === 'dark' ? 'sun' : 'moon';
  const collapseIcon = sidebarCollapsed ? 'panel-left-open' : 'panel-left-close';

  const toggleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    storage.setTheme(next);
    setTheme(next);
  };

  const handleLogout = async () => {
    const [, err] = await to(apiConfirmed('/api/v1/auth/logout', { method: 'POST' }));
    if (err instanceof ConfirmCancelledError) return;
    auth.logoutLocal();
    window.location.assign('/login');
  };

  const onResizerPointerDown = (e: React.PointerEvent<HTMLDivElement>) => {
    if (sidebarCollapsed || e.button !== 0) return;
    e.preventDefault();
    setResizing(true);
    const startX = e.clientX;
    const startWidth = sidebarWidth;

    const onMove = (ev: PointerEvent) => {
      const next = applySidebarWidth(startWidth + (ev.clientX - startX));
      storage.setSidebarWidth(next);
    };
    const onUp = () => {
      setResizing(false);
      document.removeEventListener('pointermove', onMove);
      document.removeEventListener('pointerup', onUp);
    };
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
  };

  const onResizerKey = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (sidebarCollapsed) return;
    if (e.key === 'ArrowLeft') {
      e.preventDefault();
      const next = applySidebarWidth(sidebarWidth - 8);
      storage.setSidebarWidth(next);
    } else if (e.key === 'ArrowRight') {
      e.preventDefault();
      const next = applySidebarWidth(sidebarWidth + 8);
      storage.setSidebarWidth(next);
    }
  };

  const navGroups = visibleNavGroups(permissions, user?.role);
  const bounds = getSidebarWidthBounds();

  const sidebarStyle = sidebarCollapsed
    ? {
        width: SIDEBAR_COLLAPSED_WIDTH,
        minWidth: SIDEBAR_COLLAPSED_WIDTH,
        maxWidth: SIDEBAR_COLLAPSED_WIDTH,
      }
    : { width: sidebarWidth, minWidth: sidebarWidth, maxWidth: sidebarWidth };

  return (
    <div className={`shell${resizing ? ' shell--sidebar-resizing' : ''}`}>
      <div
        className="drawer-overlay"
        style={{ zIndex: 199, display: sidebarOpen ? 'block' : 'none' }}
        aria-hidden={sidebarOpen ? 'false' : 'true'}
        onClick={closeSidebar}
      />

      <nav
        className={`sidebar${sidebarCollapsed ? ' sidebar--collapsed' : ''}${sidebarOpen ? ' sidebar--open' : ''}`}
        style={sidebarStyle}
      >
        <div className="sidebar__head">
          <Link to="/" className="sidebar__logo" title="ad-event-processor home">
            <Icon name="layers" size={32} className="sidebar__logo-icon" />
            <span className="sidebar__logo-text">
              <span className="sidebar__logo-bid">Bid</span>
              <span className="sidebar__logo-shard">Shard</span>
            </span>
          </Link>
          <SidebarSearch ref={searchRef} />
        </div>

        <div ref={sidebarBodyRef} className="sidebar__body">
          <div className="sidebar__nav">
            {navGroups.map((group) => (
              <div key={group.title} className="sidebar__group">
                <div className="sidebar__group-label">{group.title}</div>
                {group.links.map((link) => {
                  const isActive =
                    link.to === '/'
                      ? path === '/'
                      : path === link.to || path.startsWith(`${link.to}/`);
                  return (
                    <Link
                      key={link.to}
                      to={link.to}
                      className={`sidebar__link${isActive ? ' sidebar__link--active' : ''}`}
                      title={link.label}
                      aria-label={link.label}
                      onClick={closeSidebar}
                    >
                      {link.icon ? (
                        <Icon name={link.icon} size={18} className="sidebar__link-icon" />
                      ) : null}
                      <span className="sidebar__link-label">{link.label}</span>
                      {link.to === '/ops' && opsOutboxPending > 0 ? (
                        <span
                          className="sidebar__badge"
                          aria-label={`${opsOutboxPending} outbox pending`}
                        >
                          {String(opsOutboxPending)}
                        </span>
                      ) : null}
                    </Link>
                  );
                })}
              </div>
            ))}
          </div>
        </div>

        <div className="sidebar__footer">
          <div className="sidebar__account">
            {user?.email ? (
              <span className="sidebar__email" title={user.email}>
                {user.email}
              </span>
            ) : null}
            {version ? <span className="sidebar__version">{`v${version}`}</span> : null}
          </div>
          <div className="sidebar__actions">
            <button
              type="button"
              className="sidebar__action-btn"
              onClick={() => applyCollapsed(!sidebarCollapsed)}
              title={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
              aria-expanded={sidebarCollapsed ? 'false' : 'true'}
            >
              <Icon name={collapseIcon} size={15} />
              <span className="sidebar__action-label">
                {sidebarCollapsed ? 'Expand' : 'Collapse'}
              </span>
            </button>
            <button
              type="button"
              className="sidebar__action-btn"
              onClick={toggleTheme}
              title="Toggle color theme"
            >
              <Icon name={themeIcon} size={15} />
              <span className="sidebar__action-label">{theme === 'dark' ? 'Light' : 'Dark'}</span>
            </button>
            <button
              type="button"
              className="sidebar__action-btn sidebar__action-btn--muted"
              onClick={() => void handleLogout()}
              title="Logout"
            >
              <Icon name="log-out" size={15} />
              <span className="sidebar__action-label">Logout</span>
            </button>
          </div>
        </div>

        <div
          className="sidebar__resizer"
          role="separator"
          aria-orientation="vertical"
          aria-label="Resize sidebar"
          aria-valuemin={bounds.min}
          aria-valuemax={bounds.max}
          aria-valuenow={sidebarWidth}
          tabIndex={0}
          onPointerDown={onResizerPointerDown}
          onDoubleClick={(e) => {
            e.preventDefault();
            if (sidebarCollapsed) return;
            const next = applySidebarWidth(SIDEBAR_WIDTH_DEFAULT);
            storage.setSidebarWidth(next);
          }}
          onKeyDown={onResizerKey}
        />
      </nav>

      <main className="main">
        <div className="main__toolbar">
          <button
            ref={hamburgerRef}
            type="button"
            className="hamburger"
            aria-label="Menu"
            aria-expanded={sidebarOpen}
            onClick={openSidebar}
          >
            ☰
          </button>
        </div>
        <div className="main__content">
          <div className="banner-slot">
            <ShellStatusBanners />
            <IdempotencyRecoveryBanner />
            <VersionBanner serverVersion={version} />
            <BootstrapBanner bootstrapComplete={bootstrapComplete} />
            <LicenseBanner license={license} supportUrl={supportUrl} />
          </div>
          <div id="app-outlet">{children}</div>
        </div>
      </main>
    </div>
  );
}
