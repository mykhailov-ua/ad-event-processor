import { useEffect, type ReactNode } from 'react';
import { installCustomScrollbars } from '../lib/custom_scrollbars.js';
import { installErrorSurface } from '../lib/error_surface.js';
import { ConfirmProvider } from './confirm_provider.js';
import { ToastStack } from './toast_stack.js';

export function AppProviders({ children }: { children: ReactNode }) {
  useEffect(() => {
    const root = document.getElementById('root');
    if (root) installErrorSurface(root);
    const scrollbars = installCustomScrollbars();
    return () => scrollbars.destroy();
  }, []);

  return (
    <ConfirmProvider>
      {children}
      <ToastStack />
    </ConfirmProvider>
  );
}
