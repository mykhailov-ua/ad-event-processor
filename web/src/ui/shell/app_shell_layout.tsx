import { useEffect, useRef, useState, type ReactNode } from 'react';
import { AppSidebar, SIDEBAR_COLLAPSED_WIDTH } from './app_sidebar.js';
import { ShellLayout } from './shell_layout.js';
import shellStyles from './app_shell_layout.module.css';
import * as storage from '../../helpers/storage.js';

export type AppShellLayoutProps = {
  children: ReactNode;
};

export function AppShellLayout({ children }: AppShellLayoutProps) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(() => storage.getSidebarCollapsed());
  const [sidebarWidth, setSidebarWidth] = useState(() => storage.getSidebarWidth());
  const [resizing, setResizing] = useState(false);
  const hamburgerRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (sidebarCollapsed) {
      document.documentElement.style.setProperty('--sidebar-width', `${SIDEBAR_COLLAPSED_WIDTH}px`);
    } else {
      document.documentElement.style.setProperty('--sidebar-width', `${sidebarWidth}px`);
    }
  }, [sidebarCollapsed, sidebarWidth]);

  useEffect(() => {
    if (!sidebarOpen) return undefined;
    const onKey = (e: globalThis.KeyboardEvent) => {
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
    const firstLink = document.querySelector('[data-shell-nav-link]');
    if (firstLink instanceof HTMLElement) firstLink.focus();
  };

  return (
    <>
      {sidebarOpen ? (
        <button
          type="button"
          className={shellStyles.overlay}
          aria-label="Close navigation menu"
          onClick={closeSidebar}
        />
      ) : null}

      <ShellLayout
        shellClassName={resizing ? shellStyles.shellResizing : undefined}
        sidebarClassName={sidebarOpen ? shellStyles.sidebarOpen : undefined}
        sidebar={
          <AppSidebar
            collapsed={sidebarCollapsed}
            width={sidebarWidth}
            onCollapsedChange={setSidebarCollapsed}
            onWidthChange={setSidebarWidth}
            onNavigate={closeSidebar}
            onResizingChange={setResizing}
          />
        }
      >
        <div className={shellStyles.mainInner}>
          <div className={shellStyles.toolbar}>
            <button
              ref={hamburgerRef}
              type="button"
              className={shellStyles.hamburger}
              aria-label="Menu"
              aria-expanded={sidebarOpen}
              onClick={openSidebar}
            >
              =
            </button>
          </div>
          <div className={shellStyles.content}>
            <div id="app-outlet" className={shellStyles.outlet}>
              {children}
            </div>
          </div>
        </div>
      </ShellLayout>
    </>
  );
}
