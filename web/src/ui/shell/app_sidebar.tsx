import { useCallback, useEffect, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { to } from '../../lib/to.js';
import { api } from '../../helpers/api_client.js';
import * as auth from '../../helpers/auth.js';
import * as storage from '../../helpers/storage.js';
import {
  clampSidebarWidth,
  getSidebarWidthBounds,
  SIDEBAR_COLLAPSED_WIDTH,
  SIDEBAR_WIDTH_DEFAULT,
} from '../../helpers/sidebar_layout.js';
import { visibleNavGroups } from '../../helpers/nav_config.js';
import { NavIcon } from './nav_icon.js';
import { cn } from '../../lib/cn.js';
import styles from './app_sidebar.module.css';

export type AppSidebarProps = {
  collapsed: boolean;
  width: number;
  onCollapsedChange: (collapsed: boolean) => void;
  onWidthChange: (width: number) => void;
  onNavigate?: () => void;
  onResizingChange?: (resizing: boolean) => void;
};

export function AppSidebar({
  collapsed,
  width,
  onCollapsedChange,
  onWidthChange,
  onNavigate,
  onResizingChange,
}: AppSidebarProps) {
  const location = useLocation();
  const path = location.pathname;

  const [version, setVersion] = useState<string | null>(null);
  const [theme, setTheme] = useState(() => storage.getTheme());

  const sidebarBodyRef = useRef<HTMLDivElement>(null);
  const scrollRatioRef = useRef(0);

  const user = auth.getUser();
  const permissions = user?.permissions ?? [];
  const navGroups = visibleNavGroups(permissions, user?.role ?? '');
  const bounds = getSidebarWidthBounds();
  const themeIcon = theme === 'dark' ? 'sun' : 'moon';
  const collapseIcon = collapsed ? 'panel-left-open' : 'panel-left-close';

  useEffect(() => {
    void api('/api/v1/meta')
      .then((res) => {
        const data = res?.data as { version?: string } | null | undefined;
        if (data?.version) setVersion(data.version);
      })
      .catch(() => {});
  }, []);

  const applyCollapsed = useCallback(
    (next: boolean) => {
      const body = sidebarBodyRef.current;
      if (body) {
        const currentMax = body.scrollHeight - body.clientHeight;
        if (currentMax > 0) scrollRatioRef.current = body.scrollTop / currentMax;
      }
      onCollapsedChange(next);
      storage.setSidebarCollapsed(next);
      requestAnimationFrame(() => {
        const el = sidebarBodyRef.current;
        if (!el) return;
        const newMax = el.scrollHeight - el.clientHeight;
        if (newMax > 0) el.scrollTop = Math.round(scrollRatioRef.current * newMax);
      });
    },
    [onCollapsedChange]
  );

  const toggleTheme = () => {
    const next = theme === 'dark' ? 'light' : 'dark';
    storage.setTheme(next);
    setTheme(next);
  };

  const handleLogout = async () => {
    const csrf = auth.getCsrfToken();
    const headers = new Headers();
    if (csrf) headers.set('X-CSRF-Token', csrf);
    await to(
      fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'same-origin',
        headers,
      })
    );
    auth.logoutLocal();
    window.location.assign('/login');
  };

  const applySidebarWidth = useCallback(
    (px: number) => {
      const next = clampSidebarWidth(px);
      onWidthChange(next);
      if (!collapsed) {
        document.documentElement.style.setProperty('--sidebar-width', `${next}px`);
      }
      return next;
    },
    [collapsed, onWidthChange]
  );

  const onResizerPointerDown = (e: PointerEvent<HTMLDivElement>) => {
    if (collapsed || e.button !== 0) return;
    e.preventDefault();
    onResizingChange?.(true);
    const startX = e.clientX;
    const startWidth = width;

    const onMove = (ev: globalThis.PointerEvent) => {
      const next = applySidebarWidth(startWidth + (ev.clientX - startX));
      storage.setSidebarWidth(next);
    };
    const onUp = () => {
      onResizingChange?.(false);
      document.removeEventListener('pointermove', onMove);
      document.removeEventListener('pointerup', onUp);
    };
    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
  };

  const onResizerKey = (e: KeyboardEvent<HTMLDivElement>) => {
    if (collapsed) return;
    if (e.key === 'ArrowLeft') {
      e.preventDefault();
      const next = applySidebarWidth(width - 8);
      storage.setSidebarWidth(next);
    } else if (e.key === 'ArrowRight') {
      e.preventDefault();
      const next = applySidebarWidth(width + 8);
      storage.setSidebarWidth(next);
    }
  };

  return (
    <div className={cn(styles.root, collapsed ? styles.collapsed : '')}>
      <div className={styles.head}>
        <Link to="/customers" className={styles.logo} title="ad-event-processor home" onClick={onNavigate}>
          <span className={styles.logoIcon}>
            <NavIcon name="layers" size={32} />
          </span>
          <span className={styles.logoText}>ad-event-processor</span>
        </Link>
      </div>

      <div ref={sidebarBodyRef} className={styles.body}>
        <nav className={styles.nav} aria-label="Main">
          {navGroups.map((group) => (
            <div key={group.title} className={styles.group}>
              <div className={styles.groupLabel}>{group.title}</div>
              {group.links.map((link) => {
                const isActive =
                  link.to === '/'
                    ? path === '/'
                    : path === link.to || path.startsWith(`${link.to}/`);
                return (
                  <Link
                    key={link.to}
                    to={link.to}
                    data-shell-nav-link=""
                    className={cn(styles.link, isActive ? styles.linkActive : '')}
                    title={link.label}
                    aria-label={link.label}
                    aria-current={isActive ? 'page' : undefined}
                    onClick={onNavigate}
                  >
                    {link.icon ? (
                      <span className={styles.linkIcon}>
                        <NavIcon name={link.icon} size={18} />
                      </span>
                    ) : null}
                    <span className={styles.linkLabel}>{link.label}</span>
                  </Link>
                );
              })}
            </div>
          ))}
        </nav>
      </div>

      <div className={styles.footer}>
        <div className={styles.account}>
          {user?.email ? (
            <span className={styles.email} title={user.email}>
              {user.email}
            </span>
          ) : null}
          {version ? <span className={styles.version}>{`v${version}`}</span> : null}
        </div>
        <div className={styles.actions}>
          <button
            type="button"
            className={styles.actionBtn}
            onClick={() => applyCollapsed(!collapsed)}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            aria-expanded={collapsed ? 'false' : 'true'}
          >
            <NavIcon name={collapseIcon} size={15} />
            <span className={styles.actionLabel}>{collapsed ? 'Expand' : 'Collapse'}</span>
          </button>
          <button
            type="button"
            className={styles.actionBtn}
            onClick={toggleTheme}
            title="Toggle color theme"
          >
            <NavIcon name={themeIcon} size={15} />
            <span className={styles.actionLabel}>{theme === 'dark' ? 'Light' : 'Dark'}</span>
          </button>
          <button
            type="button"
            className={cn(styles.actionBtn, styles.actionBtnMuted)}
            onClick={() => void handleLogout()}
            title="Logout"
          >
            <NavIcon name="log-out" size={15} />
            <span className={styles.actionLabel}>Logout</span>
          </button>
        </div>
      </div>

      <div
        className={styles.resizer}
        role="separator"
        aria-orientation="vertical"
        aria-label="Resize sidebar"
        aria-valuemin={bounds.min}
        aria-valuemax={bounds.max}
        aria-valuenow={width}
        tabIndex={collapsed ? -1 : 0}
        hidden={collapsed}
        onPointerDown={onResizerPointerDown}
        onDoubleClick={(e) => {
          e.preventDefault();
          if (collapsed) return;
          const next = applySidebarWidth(SIDEBAR_WIDTH_DEFAULT);
          storage.setSidebarWidth(next);
        }}
        onKeyDown={onResizerKey}
      />
    </div>
  );
}

export { SIDEBAR_COLLAPSED_WIDTH };
